package dashboard

import (
	"fmt"
	"maps"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

type postmortemMode int

const (
	postRead postmortemMode = iota
	postEdit
	postDiscard
)

type postmortem struct {
	name string
	// generated is the write-up as built from the incident, so a save that
	// changed nothing does not pin a stale copy over later ones.
	generated string
	md        string
	saved     string
	rec       []byte
	mode      postmortemMode
	w, h      int
	palette   string
	page      viewport.Model
	editor    textarea.Model
}

func (m Model) openPostmortem(name, md string, rec []byte) Model {
	editor := textarea.New()
	editor.CharLimit, editor.MaxHeight, editor.ShowLineNumbers, editor.Prompt = 0, 0, false, ""
	editor.Cursor.SetMode(cursor.CursorStatic)
	editor.FocusedStyle.CursorLine = lipgloss.NewStyle()
	editor.FocusedStyle.Text = lipgloss.NewStyle().Foreground(colorText)
	generated := md
	if draft, ok := m.drafts[name]; ok {
		md = draft
	}
	m.post = &postmortem{name: name, generated: generated, md: md, saved: md, rec: rec, editor: editor}
	return m.fitPostmortem()
}

func (m Model) postmortemSize() (int, int) {
	return max(m.width, 10), max(m.bodyHeight()-1, 1)
}

func (m Model) fitPostmortem() Model {
	w, h := m.postmortemSize()
	p := *m.post
	if p.w == w && p.h == h && p.palette == paletteName {
		return m
	}
	p.w, p.h, p.palette = w, h, paletteName
	p.editor.SetWidth(w)
	p.editor.SetHeight(h)
	p.page.Width, p.page.Height = w, h
	p.page.SetContent(renderMarkdown(p.md, w))
	m.post = &p
	return m
}

func (m Model) handlePostmortemKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	p := *m.post
	switch key := msg.String(); {
	case key == "ctrl+c":
		return m.closeLogs(), tea.Quit
	case key == "ctrl+s":
		return m.savePostmortem()
	case p.mode == postDiscard:
		if key == "y" {
			m.post = nil
			return m, nil
		}
		p.mode = postRead
	case p.mode == postEdit:
		if key == "esc" {
			p.md, p.mode = p.editor.Value(), postRead
			p.editor.Blur()
			p.page.SetContent(renderMarkdown(p.md, p.w))
			break
		}
		var cmd tea.Cmd
		p.editor, cmd = p.editor.Update(msg)
		m.post = &p
		return m, cmd
	default:
		return m.handlePostmortemReadKey(p, msg)
	}
	m.post = &p
	return m, nil
}

func (m Model) handlePostmortemReadKey(p postmortem, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if p.md == p.saved {
			m.post = nil
			return m, nil
		}
		p.mode = postDiscard
	case "e":
		p.mode = postEdit
		p.editor.SetValue(p.md)
		p.editor.Focus()
		p.editor, _ = p.editor.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
	case "g", "home":
		p.page.GotoTop()
	case "G", "end":
		p.page.GotoBottom()
	default:
		p.page, _ = p.page.Update(msg)
	}
	m.post = &p
	return m, nil
}

func (m Model) savePostmortem() (Model, tea.Cmd) {
	p := *m.post
	switch p.mode {
	case postEdit:
		p.md = p.editor.Value()
	case postDiscard:
		p.mode = postRead
	case postRead:
	}
	m.post = &p
	write := m.cfg.WriteFile
	if write == nil {
		return m.say("nowhere to save the write-up"), nil
	}
	name, md, rec, draft := p.name, p.md, p.rec, p.md != p.generated
	return m, func() tea.Msg {
		if err := write(name+".json", rec); err != nil {
			return writtenMsg{path: name + ".json", err: err}
		}
		return writtenMsg{path: name + ".md", err: write(name+".md", []byte(md)), name: name, md: md, draft: draft}
	}
}

func (m Model) keepSaved(msg writtenMsg) Model {
	if msg.err != nil || msg.name == "" {
		return m
	}
	m.drafts = maps.Clone(m.drafts)
	if m.drafts == nil {
		m.drafts = map[string]string{}
	}
	delete(m.drafts, msg.name)
	if msg.draft {
		m.drafts[msg.name] = msg.md
	}
	if m.post != nil && m.post.name == msg.name {
		p := *m.post
		p.saved = msg.md
		m.post = &p
	}
	return m
}

func (p *postmortem) dirty() bool {
	if p.mode == postEdit {
		return p.editor.Value() != p.saved
	}
	return p.md != p.saved
}

func (m Model) postmortemBody() string {
	p := m.post
	title := styleHeading.Render("WRITE-UP") + " " + styleTitle.Render(p.name+".md")
	switch {
	case p.mode == postDiscard:
		title += "  " + styleBoldRed.Render("discard changes? y/n")
	case p.dirty():
		title += "  " + styleUpgrade.Render("● unsaved")
	}
	if p.mode == postEdit {
		return title + "\n" + p.editor.View()
	}
	if p.page.TotalLineCount() > p.h {
		title += styleFaint.Render(fmt.Sprintf("  %3.f%%", p.page.ScrollPercent()*100))
	}
	return title + "\n" + p.page.View()
}

func (p *postmortem) keys() [][2]string {
	switch p.mode {
	case postEdit:
		return [][2]string{{"ctrl+s", "save"}, {"esc", "preview"}}
	case postDiscard:
		return [][2]string{{"y", "discard"}, {"n", "keep editing"}}
	case postRead:
	}
	return [][2]string{{"↑↓ pgup pgdn", "scroll"}, {"g G", "top/bottom"}, {"e", "edit"}, {"ctrl+s", "save"}, {"q", "close"}}
}

func renderMarkdown(md string, w int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle()),
		glamour.WithWordWrap(max(w-2, 1)),
		glamour.WithColorProfile(lipgloss.ColorProfile()),
	)
	if err != nil {
		return md
	}
	out, err := r.Render(md)
	if err != nil {
		return md
	}
	return indent(strings.Trim(out, "\n"), " ")
}

func markdownStyle() ansi.StyleConfig {
	s := styles.DarkStyleConfig
	if !lightColor(colorText) {
		s = styles.LightStyleConfig
	}
	text, accent, ink := string(colorText), string(colorTeal), string(colorInk)
	// glamour wraps tables to the word wrap and then adds the margin, which
	// pushes their last column past the body.
	var noMargin uint
	s.Document.Color, s.Document.Margin = &text, &noMargin
	s.Heading.Color = &accent
	s.H1.Color, s.H1.BackgroundColor = &ink, &accent
	s.Link.Color, s.LinkText.Color = &accent, &accent
	return s
}

func lightColor(c lipgloss.Color) bool {
	var r, g, b uint8
	if _, err := fmt.Sscanf(string(c), "#%02x%02x%02x", &r, &g, &b); err != nil {
		return true
	}
	return 0.2126*float64(r)+0.7152*float64(g)+0.0722*float64(b) > 128
}
