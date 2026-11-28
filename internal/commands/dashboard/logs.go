package dashboard

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
	"github.com/mattn/go-runewidth"
)

const (
	logsKept       = 500
	defaultLogPoll = 2 * time.Second
)

// LogBatch is one flush of a deployment's logs.
type LogBatch = deployments.LogBatch

// DeploymentLog is a deployment's status and every batch of logs it has
// flushed so far, oldest first.
type DeploymentLog struct {
	Status  string
	Batches []LogBatch
}

// DeploymentService streams a deployment's logs as they are flushed.
type DeploymentService interface {
	StreamLogs(ctx context.Context, deploymentID string) (*streaming.Stream[LogBatch], error)
}

const deploymentLogsQuery = `query dashboardDeploymentLogs($organizationId: ID!, $id: UUID!) {
  deployment(organizationId: $organizationId, id: $id) { status logs { timestamp message } }
}`

func (a *gqlAPI) DeploymentLogs(ctx context.Context, deploymentID string) (DeploymentLog, error) {
	var data struct {
		Deployment struct {
			Status string     `json:"status"`
			Logs   []LogBatch `json:"logs"`
		} `json:"deployment"`
	}
	vars := map[string]any{"organizationId": a.orgID, "id": deploymentID}
	if err := a.request(ctx, "dashboardDeploymentLogs", deploymentLogsQuery, vars, &data); err != nil {
		return DeploymentLog{}, err
	}
	return DeploymentLog{Status: data.Deployment.Status, Batches: data.Deployment.Logs}, nil
}

func (a *gqlAPI) StreamDeploymentLogs(ctx context.Context, deploymentID string) (*streaming.Stream[LogBatch], error) {
	if a.deploys == nil {
		return nil, streaming.ErrRequiresPAT
	}
	return a.deploys.StreamLogs(ctx, deploymentID)
}

type (
	logsSubscribedMsg struct {
		gen    int
		stream *streaming.Stream[LogBatch]
		err    error
	}
	logsReadMsg struct {
		gen int
		log DeploymentLog
		err error
	}
	logBatchesMsg struct {
		gen     int
		batches []LogBatch
	}
	logsClosedMsg struct{ gen int }
	logsPollMsg   struct{ gen, tick int }
)

type logsMsg interface{ logsGen() int }

func (m logsSubscribedMsg) logsGen() int { return m.gen }
func (m logsReadMsg) logsGen() int       { return m.gen }
func (m logBatchesMsg) logsGen() int     { return m.gen }
func (m logsClosedMsg) logsGen() int     { return m.gen }
func (m logsPollMsg) logsGen() int       { return m.gen }

// logView follows one deployment's logs in the side panel. gen tells its
// messages from those of a deployment followed before.
type logView struct {
	open bool
	gen  int
	// tab is where the logs show: the side panel on the map, or the whole
	// body on the Deployments tab.
	tab    Tab
	env    string
	inst   string
	deploy string
	action string
	status string
	lines  []string
	// back is how many lines the view sits above the newest; zero follows
	// the tail.
	back int
	// last is the newest batch shown, and atLast the messages shown with
	// that timestamp, so a batch both read and streamed shows once. Logs carry
	// no ids, so two identical lines in one instant also show once.
	last      time.Time
	atLast    []string
	ctx       context.Context //nolint:containedctx // reads and the stream outlive any one update
	cancel    context.CancelFunc
	stream    *streaming.Stream[LogBatch]
	receiving bool
	polling   bool
	// tick keeps one poll timer running: scheduling another stops the last.
	tick int
	err  error
}

func (l logView) done() bool { return deployments.IsTerminal(l.status) }

func (m Model) toggleLogs() (Model, tea.Cmd) {
	if m.logs.open {
		return m.closeLogs(), nil
	}
	return m.openLogs()
}

func (m Model) closeLogs() Model {
	if m.logs.cancel != nil {
		m.logs.cancel()
	}
	m.logs = logView{gen: m.logs.gen}
	return m
}

// followed is the instance whose deployment is worth reading in an
// environment: one in flight, or else the newest.
func followed(env *EnvNode) *InstNode {
	if deploying := env.Deploying(); len(deploying) > 0 {
		return deploying[0]
	}
	var best *InstNode
	for _, inst := range env.Instances {
		if inst.Deploy == nil || inst.Deploy.ID == "" {
			continue
		}
		if best == nil || inst.Deploy.CreatedAt > best.Deploy.CreatedAt {
			best = inst
		}
	}
	return best
}

func (m Model) openLogs() (Model, tea.Cmd) {
	var inst *InstNode
	if env := m.graph.Env(m.selected); env != nil && !env.Restricted {
		inst = followed(env)
	}
	if inst == nil {
		m.toast, m.toastAt = "no deployment to follow here", m.frame
		return m, nil
	}
	return m.follow(TabConstellation, m.selected, inst.ID, *inst.Deploy)
}

// logsInView reports whether the followed logs are still what the screen is
// about.
func (m Model) logsInView() bool {
	switch {
	case m.tab != m.logs.tab:
		return false
	case m.tab == TabDeployments:
		return m.desk.selected == m.logs.deploy
	}
	return m.selected == m.logs.env
}

func (m Model) follow(tab Tab, env, inst string, d Deployment) (Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(m.ctx())
	m.logs = logView{
		open: true, gen: m.logs.gen + 1, tab: tab, env: env, inst: inst, deploy: d.ID,
		action: d.Action, status: d.Status, ctx: ctx, cancel: cancel,
	}
	if m.logs.done() {
		return m, m.readLogs()
	}
	api, id, gen := m.api, m.logs.deploy, m.logs.gen
	return m, func() tea.Msg {
		stream, err := api.StreamDeploymentLogs(ctx, id)
		return logsSubscribedMsg{gen: gen, stream: stream, err: err}
	}
}

func (m Model) readLogs() tea.Cmd {
	api, ctx, id, gen := m.api, m.logs.ctx, m.logs.deploy, m.logs.gen
	return func() tea.Msg {
		log, err := api.DeploymentLogs(ctx, id)
		return logsReadMsg{gen: gen, log: log, err: err}
	}
}

func (m Model) schedulePoll() (Model, tea.Cmd) {
	m.logs.tick++
	return m, after(m.cfg.LogPoll, logsPollMsg{gen: m.logs.gen, tick: m.logs.tick})
}

func (m Model) updateLogs(msg logsMsg) (Model, tea.Cmd) {
	if !m.logs.open || msg.logsGen() != m.logs.gen {
		return m, nil
	}
	switch msg := msg.(type) {
	case logsSubscribedMsg:
		if msg.err != nil {
			m.logs.polling = true
			return m, m.readLogs()
		}
		m.logs.stream = msg.stream
		// The read after subscribing fills in what was flushed before the
		// stream opened; the stream is drained only once it lands.
		return m, m.readLogs()
	case logsReadMsg:
		return m.applyLogsRead(msg)
	case logBatchesMsg:
		m.logs = m.logs.add(msg.batches)
		if m.logs.stream == nil {
			return m, nil
		}
		return m, receiveLogs(m.logs.gen, m.logs.stream)
	case logsClosedMsg:
		m.logs.stream, m.logs.receiving = nil, false
		if m.logs.done() {
			return m, nil
		}
		m.logs.polling = true
		m.logs.tick++
		return m, m.readLogs()
	case logsPollMsg:
		if msg.tick != m.logs.tick || m.logs.done() {
			return m, nil
		}
		if m.logs.polling || m.endedOnMap() {
			return m, m.readLogs()
		}
		return m.schedulePoll()
	}
	return m, nil
}

// endedOnMap reports whether the last read of the organization shows the
// followed deployment over, so a stream that stays open after it ends does
// not keep the view waiting.
func (m Model) endedOnMap() bool {
	if m.graph == nil {
		return false
	}
	inst := m.graph.Instance(m.logs.inst)
	if inst == nil || inst.Deploy == nil {
		return false
	}
	return inst.Deploy.ID != m.logs.deploy || deployments.IsTerminal(inst.Deploy.Status)
}

func (m Model) applyLogsRead(msg logsReadMsg) (Model, tea.Cmd) {
	m.logs.err = msg.err
	if msg.err == nil {
		m.logs = m.logs.add(msg.log.Batches)
		if msg.log.Status != "" {
			m.logs.status = msg.log.Status
		}
	}
	if m.logs.done() {
		m.logs.cancel()
		m.logs.stream, m.logs.receiving = nil, false
		return m, nil
	}
	var cmds []tea.Cmd
	if m.logs.stream != nil && !m.logs.receiving {
		m.logs.receiving = true
		cmds = append(cmds, receiveLogs(m.logs.gen, m.logs.stream))
	}
	m, poll := m.schedulePoll()
	return m, tea.Batch(append(cmds, poll)...)
}

func receiveLogs(gen int, stream *streaming.Stream[LogBatch]) tea.Cmd {
	return func() tea.Msg {
		batch, ok := <-stream.C
		if !ok {
			return logsClosedMsg{gen: gen}
		}
		batches := []LogBatch{batch}
		for len(batches) < eventBatch {
			select {
			case batch, ok := <-stream.C:
				if !ok {
					return logBatchesMsg{gen: gen, batches: batches}
				}
				batches = append(batches, batch)
			default:
				return logBatchesMsg{gen: gen, batches: batches}
			}
		}
		return logBatchesMsg{gen: gen, batches: batches}
	}
}

func (l logView) add(batches []LogBatch) logView {
	var fresh []string
	for _, b := range batches {
		switch {
		case b.Timestamp.Before(l.last):
			continue
		case b.Timestamp.Equal(l.last):
			if slices.Contains(l.atLast, b.Message) {
				continue
			}
			l.atLast = append(slices.Clip(l.atLast), b.Message)
		default:
			l.last, l.atLast = b.Timestamp, []string{b.Message}
		}
		fresh = append(fresh, logLines(b.Message)...)
	}
	if len(fresh) == 0 {
		return l
	}
	lines := slices.Concat(l.lines, fresh)
	l.lines = lines[max(len(lines)-logsKept, 0):]
	if l.back > 0 {
		l.back = min(l.back+len(fresh), len(l.lines)-1)
	}
	return l
}

func (m Model) scrollLogs(key string) (Model, bool) {
	page := max(m.bodyHeight()-6, 4)
	back := m.logs.back
	switch key {
	case "pgup", "ctrl+u":
		back += page
	case "pgdown", "ctrl+d":
		back -= page
	case "shift+up":
		back++
	case "shift+down":
		back--
	case "home":
		back = len(m.logs.lines)
	case "end", "G":
		back = 0
	default:
		return m, false
	}
	m.logs.back = max(min(back, len(m.logs.lines)-1), 0)
	return m, true
}

func logLines(message string) []string {
	if message == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(message, "\n"), "\n")
	for i, line := range lines {
		lines[i] = clean(strings.ReplaceAll(ansi.Strip(line), "\t", "    "))
	}
	return lines
}

func (m Model) logsPanel(w, h int) string {
	l := m.logs
	name := strings.TrimPrefix(l.inst, l.env+"-")
	head := []string{
		styleHeading.Render("LOGS") + " " + styleTitle.Render(clip(name, w-5)),
		styleMuted.Render(strings.ToLower(l.action)+" · ") + logStatus(l.status),
	}
	if l.err != nil {
		head = append(head, styleRed.Render(clip("could not read: "+clean(l.err.Error()), w)))
	}
	if l.back > 0 {
		head = append(head, styleUpgrade.Render(fmt.Sprintf("↑ %d lines back", l.back))+styleFaint.Render(" · end follows"))
	}
	head = append(head, "")
	room := max(h-len(head), 0)
	var body []string
	for i := len(l.lines) - 1 - l.back; i >= 0 && len(body) < room; i-- {
		body = append(wrapLog(l.lines[i], w), body...)
	}
	body = body[max(len(body)-room, 0):]
	out := make([]string, 0, len(head)+len(body)+1)
	out = append(out, head...)
	if len(l.lines) == 0 {
		out = append(out, styleFaint.Render("no logs yet"))
	}
	out = append(out, body...)
	return strings.Join(out, "\n")
}

func wrapLog(line string, w int) []string {
	if w < 8 || runewidth.StringWidth(line) <= w {
		return []string{clip(line, w)}
	}
	var out []string
	first := true
	for line != "" {
		width, indent := w, ""
		if !first {
			width, indent = w-2, "  "
		}
		cut, used := 0, 0
		for i, r := range line {
			rw := runewidth.RuneWidth(r)
			if used+rw > width {
				break
			}
			cut, used = i+len(string(r)), used+rw
		}
		if cut == 0 {
			break
		}
		out = append(out, indent+line[:cut])
		line, first = line[cut:], false
	}
	return out
}

func logStatus(status string) string {
	text := strings.ToLower(status)
	switch deployments.Status(status) {
	case deployments.StatusCompleted:
		return styleTeal.Render("✓ " + text)
	case deployments.StatusFailed, deployments.StatusAborted, deployments.StatusRejected:
		return styleRed.Render("✕ " + text)
	case deployments.StatusRunning, deployments.StatusPending:
		return styleDeploy.Render("⟳ " + text)
	case deployments.StatusProposed, deployments.StatusApproved:
	}
	return styleMuted.Render(text)
}
