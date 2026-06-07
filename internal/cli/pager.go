package cli

import (
	"fmt"
	"iter"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// PagerConfig describes how a paginated result set is rendered by [Paginate].
type PagerConfig[T any] struct {
	// Columns are the table header titles.
	Columns []string
	// Row maps one item to its cells. The returned slice must align with Columns.
	Row func(T) []string
	// Out is the destination; defaults to os.Stdout when nil. Used both to
	// decide whether output is interactive and as the render target.
	Out *os.File
	// PageSize is how many rows are pulled from the iterator per batch in
	// interactive mode. Zero uses a sensible default.
	PageSize int
}

// Paginate renders a paginated iterator.
//
// When Out is an interactive terminal it launches a scrollable table that lazily
// pulls more rows as the cursor approaches the end, so a multi-thousand-row list
// never fetches more than the user actually scrolls through. The user quits with
// q / esc / ctrl-c.
//
// When Out is not a terminal (a pipe, a file, CI), it drains the iterator and
// writes a plain table — keeping scripts and output redirection working. The
// caller is responsible for routing structured (--output json) requests
// elsewhere before reaching here.
func Paginate[T any](seq iter.Seq2[T, error], cfg PagerConfig[T]) error {
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	if IsInteractive(out) {
		return runInteractivePager(seq, cfg, out)
	}
	return streamAll(seq, cfg, out)
}

// IsInteractive reports whether f is attached to a terminal. It is the single
// gate used to decide between the interactive pager and plain streaming.
func IsInteractive(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// streamAll drains the iterator into a plain table written to out. Used for
// non-interactive destinations where paging UX makes no sense.
func streamAll[T any](seq iter.Seq2[T, error], cfg PagerConfig[T], out *os.File) error {
	headers := make([]any, len(cfg.Columns))
	for i, c := range cfg.Columns {
		headers[i] = c
	}
	tbl := NewTable(headers...).WithWriter(out)
	for v, err := range seq {
		if err != nil {
			return err
		}
		cells := cfg.Row(v)
		row := make([]any, len(cells))
		for i, c := range cells {
			row[i] = c
		}
		tbl.AddRow(row...)
	}
	tbl.Print()
	return nil
}

const defaultPageSize = 25

// rowsMsg carries a freshly pulled batch from the iterator into the model's
// Update loop. rows are already item-agnostic ([]string), so the message itself
// need not be generic.
type rowsMsg struct {
	rows []table.Row
	done bool
	err  error
}

// pagerModel is the bubbletea model backing the interactive table. It owns a
// pull function (from iter.Pull2) and appends rows to the embedded table as the
// user scrolls toward the end of what's loaded.
type pagerModel[T any] struct {
	table    table.Model
	next     func() (T, error, bool)
	rowFn    func(T) []string
	columns  []string
	cols     []table.Column // applied column widths; used to truncate row cells
	pageSize int

	loaded      int
	done        bool // iterator exhausted
	loading     bool // a load batch is in flight
	colsApplied bool // column widths sized from the first batch
	quitting    bool
	err         error
}

func newPagerModel[T any](next func() (T, error, bool), cfg PagerConfig[T]) pagerModel[T] {
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	cols := make([]table.Column, len(cfg.Columns))
	for i, c := range cfg.Columns {
		cols[i] = table.Column{Title: c, Width: runewidth.StringWidth(c) + 2}
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	styles := table.DefaultStyles()
	styles.Header = styles.Header.Bold(true).Foreground(lipgloss.Color("12")).BorderBottom(true)
	styles.Selected = styles.Selected.Bold(true).Foreground(lipgloss.Color("0")).Background(lipgloss.Color("12"))
	t.SetStyles(styles)

	return pagerModel[T]{
		table:    t,
		next:     next,
		rowFn:    cfg.Row,
		columns:  cfg.Columns,
		pageSize: pageSize,
		loading:  true, // Init kicks off the first load
	}
}

func (m pagerModel[T]) Init() tea.Cmd { return m.loadCmd() }

// loadCmd pulls up to pageSize items from the iterator. It runs in a tea.Cmd
// goroutine; the model's `loading` guard ensures only one is ever in flight, so
// the underlying pull function is never called concurrently.
func (m pagerModel[T]) loadCmd() tea.Cmd {
	next, rowFn, n := m.next, m.rowFn, m.pageSize
	return func() tea.Msg {
		rows := make([]table.Row, 0, n)
		for range n {
			v, err, ok := next()
			if err != nil {
				return rowsMsg{err: err}
			}
			if !ok {
				return rowsMsg{rows: rows, done: true}
			}
			rows = append(rows, table.Row(rowFn(v)))
		}
		return rowsMsg{rows: rows}
	}
}

func (m pagerModel[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve two lines for the footer; keep a sane floor.
		h := msg.Height - 2
		if h < 3 {
			h = 3
		}
		m.table.SetHeight(h)
		return m, nil

	case rowsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.quitting = true
			return m, tea.Quit
		}
		if !m.colsApplied && len(msg.rows) > 0 {
			m.cols = sizeColumns(m.columns, msg.rows)
			m.table.SetColumns(m.cols)
			m.colsApplied = true
		}
		if len(msg.rows) > 0 {
			m.table.SetRows(append(m.table.Rows(), truncateRows(msg.rows, m.cols)...))
			m.loaded = len(m.table.Rows())
		}
		if msg.done {
			m.done = true
		}
		// Keep loading until the visible window is filled, so the first screen
		// isn't half-empty when pageSize is smaller than the terminal height.
		if !m.done && !m.loading && m.loaded < m.table.Height()+m.pageSize {
			m.loading = true
			return m, m.loadCmd()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds := []tea.Cmd{cmd}
	// Prefetch the next batch once the cursor gets within a page of the end.
	if !m.done && !m.loading && m.table.Cursor() >= m.loaded-m.pageSize {
		m.loading = true
		cmds = append(cmds, m.loadCmd())
	}
	return m, tea.Batch(cmds...)
}

var footerStyle = lipgloss.NewStyle().Faint(true)

func (m pagerModel[T]) View() string {
	if m.quitting {
		return ""
	}
	status := fmt.Sprintf("%d loaded", m.loaded)
	switch {
	case m.loading:
		status += " · loading…"
	case m.done:
		status += " · end"
	}
	if m.loaded == 0 && m.done {
		return footerStyle.Render("No results.") + "\n"
	}
	footer := footerStyle.Render(fmt.Sprintf("↑/↓ scroll · pgup/pgdn page · %s · q quit", status))
	return m.table.View() + "\n" + footer
}

// sizeColumns derives column widths from the headers and a sample batch, so the
// table is laid out from real data on the first page rather than guessed widths.
func sizeColumns(headers []string, sample []table.Row) []table.Column {
	const maxWidth = 50
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = runewidth.StringWidth(h)
	}
	for _, r := range sample {
		for i := range headers {
			if i < len(r) {
				if w := runewidth.StringWidth(r[i]); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}
	cols := make([]table.Column, len(headers))
	for i, h := range headers {
		w := widths[i]
		if w > maxWidth {
			w = maxWidth
		}
		cols[i] = table.Column{Title: h, Width: w + 2}
	}
	return cols
}

// truncateRows clips each cell to its column's data width, appending "…" when
// the value was longer. Without this, rows from later batches (or any cell wider
// than the sampled max) would overflow the column and push the table past the
// terminal edge.
func truncateRows(rows []table.Row, cols []table.Column) []table.Row {
	if len(cols) == 0 {
		return rows
	}
	out := make([]table.Row, len(rows))
	for i, r := range rows {
		nr := make(table.Row, len(r))
		for j, cell := range r {
			if j < len(cols) {
				w := cols[j].Width - 2 // strip the +2 padding sizeColumns added
				if w < 1 {
					w = 1
				}
				nr[j] = runewidth.Truncate(cell, w, "…")
			} else {
				nr[j] = cell
			}
		}
		out[i] = nr
	}
	return out
}

// runInteractivePager drives the bubbletea program. It owns the iter.Pull2
// lifetime so the underlying iterator goroutine is always released on exit.
func runInteractivePager[T any](seq iter.Seq2[T, error], cfg PagerConfig[T], out *os.File) error {
	next, stop := iter.Pull2(seq)
	defer stop()

	m := newPagerModel(next, cfg)
	p := tea.NewProgram(m, tea.WithOutput(out), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(pagerModel[T]); ok && fm.err != nil {
		return fm.err
	}
	return nil
}
