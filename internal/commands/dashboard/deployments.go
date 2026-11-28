package dashboard

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

const (
	recentDeployments   = 20
	deploymentInstances = 100
	rejectionKey        = "review/rejection"
)

var (
	activeStatuses   = []string{"PROPOSED", "PENDING", "RUNNING", "APPROVED"}
	finishedStatuses = []string{"COMPLETED", "FAILED", "ABORTED", "REJECTED"}
)

// DeploymentQuery narrows a read of deployments to some instances and
// statuses.
type DeploymentQuery struct {
	Instances []string
	Statuses  []string
}

const deploymentsQuery = `query dashboardDeployments($organizationId: ID!, $instances: [ID!]!, $statuses: [DeploymentStatus!]!, $cursor: Cursor) {
  deployments(
    organizationId: $organizationId
    filter: {instanceId: {in: $instances}, status: {in: $statuses}}
    sort: {field: CREATED_AT, order: DESC}
    cursor: $cursor
  ) {
    cursor { next }
    items { id status action version message deployedBy createdAt elapsedTime instance { id } }
  }
}`

func (a *gqlAPI) Deployments(ctx context.Context, q DeploymentQuery, cursor string) ([]Deployment, string, error) {
	cur := map[string]any{"limit": pageLimit}
	if cursor != "" {
		cur["next"] = cursor
	}
	vars := map[string]any{"organizationId": a.orgID, "instances": q.Instances, "statuses": q.Statuses, "cursor": cur}
	var data struct {
		Page page[Deployment] `json:"deployments"`
	}
	err := a.request(ctx, "dashboardDeployments", deploymentsQuery, vars, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

// DeploymentBoard is the deployments in view, each list newest first:
// those awaiting approval, those in flight, and the most recent to finish.
type DeploymentBoard struct {
	Awaiting []Deployment
	InFlight []Deployment
	Recent   []Deployment
}

// Rows lists the board in the order it is shown.
func (b DeploymentBoard) Rows() []Deployment {
	return slices.Concat(b.Awaiting, b.InFlight, b.Recent)
}

// ReadDeployments reads every unfinished deployment of the instances, and
// the most recent to finish.
func ReadDeployments(ctx context.Context, api API, instances []string) (DeploymentBoard, error) {
	var board DeploymentBoard
	var active []Deployment
	for chunk := range slices.Chunk(instances, deploymentInstances) {
		open, err := drain(ctx, func(ctx context.Context, cursor string) ([]Deployment, string, error) {
			return api.Deployments(ctx, DeploymentQuery{Instances: chunk, Statuses: activeStatuses}, cursor)
		})
		if err != nil {
			return DeploymentBoard{}, fmt.Errorf("list deployments: %w", err)
		}
		active = append(active, open...)
		// Only the newest page of history is worth reading: the board shows a
		// few, and digging further is what the web app is for.
		done, _, err := api.Deployments(ctx, DeploymentQuery{Instances: chunk, Statuses: finishedStatuses}, "")
		if err != nil {
			return DeploymentBoard{}, fmt.Errorf("list deployments: %w", err)
		}
		board.Recent = append(board.Recent, done...)
	}
	for _, d := range active {
		if d.Status == string(deployments.StatusProposed) {
			board.Awaiting = append(board.Awaiting, cleanDeployment(d))
		} else {
			board.InFlight = append(board.InFlight, cleanDeployment(d))
		}
	}
	for i := range board.Recent {
		board.Recent[i] = cleanDeployment(board.Recent[i])
	}
	for _, list := range []*[]Deployment{&board.Awaiting, &board.InFlight, &board.Recent} {
		slices.SortStableFunc(*list, newestFirst)
	}
	board.Recent = board.Recent[:min(len(board.Recent), recentDeployments)]
	return board, nil
}

func newestFirst(a, b Deployment) int {
	return cmp.Or(createdAt(b).Compare(createdAt(a)), strings.Compare(a.ID, b.ID))
}

func createdAt(d Deployment) time.Time {
	t, _ := time.Parse(time.RFC3339, d.CreatedAt)
	return t
}

func cleanDeployment(d Deployment) Deployment {
	d.Status, d.Action, d.Version = clean(d.Status), clean(d.Action), clean(d.Version)
	d.Message, d.DeployedBy = cleanLines(d.Message), clean(d.DeployedBy)
	return d
}

func instanceOf(d Deployment) string {
	if d.Instance == nil {
		return ""
	}
	return d.Instance.ID
}

// DeploymentPeek is a deployment's configuration and what it changes.
type DeploymentPeek struct {
	Deployment types.Deployment
	// Baseline is the instance's last completed provision before the
	// deployment, which Version and Params compare against. Without one,
	// every param shows as new.
	Baseline *Deployment
	Version  types.VersionComparison
	Params   []types.ParamComparison
}

func (a *gqlAPI) PeekDeployment(ctx context.Context, instanceID, deploymentID string) (DeploymentPeek, error) {
	if a.deployer == nil {
		return DeploymentPeek{}, ErrNoDeployer
	}
	d, err := a.deployer.GetDeployment(ctx, deploymentID)
	if err != nil {
		return DeploymentPeek{}, err
	}
	peek := DeploymentPeek{Deployment: *d, Version: types.VersionComparison{Target: d.Version}}
	base, ok, err := a.baseline(ctx, instanceID, d)
	if err != nil {
		return DeploymentPeek{}, err
	}
	if !ok {
		peek.Params = flattenParams(d.Params)
		return peek, nil
	}
	diff, err := a.deployer.CompareDeployments(ctx, base.ID, d.ID)
	if err != nil {
		return DeploymentPeek{}, err
	}
	peek.Baseline, peek.Version, peek.Params = &base, diff.Version, diff.Params
	return peek, nil
}

func (a *gqlAPI) baseline(ctx context.Context, instanceID string, d *types.Deployment) (Deployment, bool, error) {
	done, _, err := a.Deployments(ctx, DeploymentQuery{Instances: []string{instanceID}, Statuses: []string{string(deployments.StatusCompleted)}}, "")
	if err != nil {
		return Deployment{}, false, err
	}
	for _, c := range done {
		if c.ID != d.ID && c.Action == string(deployments.ActionProvision) && (d.CreatedAt.IsZero() || createdAt(c).Before(d.CreatedAt)) {
			return c, true, nil
		}
	}
	return Deployment{}, false, nil
}

// flattenParams lists params leaf by leaf, as a comparison against nothing.
func flattenParams(params map[string]any) []types.ParamComparison {
	var out []types.ParamComparison
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch v := v.(type) {
		case map[string]any:
			if len(v) == 0 {
				out = append(out, newParam(path, "{}"))
			}
			for _, k := range slices.Sorted(maps.Keys(v)) {
				walk(path+"."+k, v[k])
			}
		case []any:
			if len(v) == 0 {
				out = append(out, newParam(path, "[]"))
			}
			for i, item := range v {
				walk(path+"["+strconv.Itoa(i)+"]", item)
			}
		case string:
			out = append(out, newParam(path, v))
		default:
			text, _ := json.Marshal(v)
			out = append(out, newParam(path, string(text)))
		}
	}
	for _, k := range slices.Sorted(maps.Keys(params)) {
		walk("."+k, params[k])
	}
	return out
}

func newParam(path, value string) types.ParamComparison {
	return types.ParamComparison{Path: path, Target: types.ParamValue{Present: true, Value: value}}
}

// ErrReasonNotSaved means a deployment was rejected, but the reason given
// for it could not be kept.
var ErrReasonNotSaved = errors.New("the reason was not saved")

func (a *gqlAPI) RejectDeployment(ctx context.Context, deploymentID, reason string) error {
	if a.deployer == nil {
		return ErrNoDeployer
	}
	var saveErr error
	if reason != "" {
		scope := stash.Scope{Kind: stash.ScopeDeployment, ID: deploymentID}
		_, saveErr = a.deployer.SetStash(ctx, scope, rejectionKey, stash.SetInput{Value: reason, Format: stash.FormatText})
	}
	// A reason that could not be kept is no reason to leave the deployment
	// waiting on someone who has decided against it.
	if _, err := a.deployer.RejectDeployment(ctx, deploymentID); err != nil {
		return err
	}
	if saveErr != nil {
		return fmt.Errorf("%w: %s", ErrReasonNotSaved, saveErr.Error())
	}
	return nil
}

type (
	deskMsg struct {
		seq   int
		board DeploymentBoard
		err   error
	}
	peekMsg struct {
		id   string
		peek DeploymentPeek
		err  error
	}
)

// deskState is the Deployments tab. The selection follows a deployment by
// id, so a refresh that reorders the board keeps it.
type deskState struct {
	board    DeploymentBoard
	read     bool
	err      error
	seq      int
	selected string
	idx      int
}

func (d deskState) selection() (Deployment, bool) {
	rows := d.board.Rows()
	if d.idx < 0 || d.idx >= len(rows) {
		return Deployment{}, false
	}
	return rows[d.idx], true
}

func (d deskState) reselect() deskState {
	rows := d.board.Rows()
	if i := slices.IndexFunc(rows, func(r Deployment) bool { return r.ID == d.selected }); i >= 0 {
		d.idx = i
		return d
	}
	d.idx = max(min(d.idx, len(rows)-1), 0)
	d.selected = ""
	if len(rows) > 0 {
		d.selected = rows[d.idx].ID
	}
	return d
}

func (d deskState) step(key string) deskState {
	rows := d.board.Rows()
	d.idx = step(d.idx, len(rows), key)
	if len(rows) > 0 {
		d.selected = rows[d.idx].ID
	}
	return d
}

type peekView struct {
	id      string
	inst    string
	env     string
	loading bool
	peek    DeploymentPeek
	err     error
	scroll  int
}

func (m Model) updateDesk(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deskMsg:
		if msg.seq != m.desk.seq {
			return m, nil
		}
		m.desk.err = msg.err
		if msg.err == nil {
			m.desk.board, m.desk.read = msg.board, true
		}
		m.desk = m.desk.reselect()
	case peekMsg:
		if m.peek == nil || m.peek.id != msg.id {
			return m, nil
		}
		p := *m.peek
		p.loading, p.peek, p.err = false, msg.peek, msg.err
		m.peek = &p
	}
	return m, nil
}

func (m Model) instancesInView() []string {
	var ids []string
	for _, env := range m.graph.Envs {
		if env.Restricted || env.Unloaded {
			continue
		}
		for _, inst := range env.Instances {
			ids = append(ids, inst.ID)
		}
	}
	return ids
}

func (m Model) fetchDesk() (Model, tea.Cmd) {
	if m.graph == nil {
		return m, nil
	}
	m.desk.seq++
	api, ctx, seq, ids := m.api, m.ctx(), m.desk.seq, m.instancesInView()
	return m, func() tea.Msg {
		board, err := ReadDeployments(ctx, api, ids)
		return deskMsg{seq: seq, board: board, err: err}
	}
}

// refreshDesk reads deployments again with the map, only while they are on
// screen.
func (m Model) refreshDesk() (Model, tea.Cmd) {
	if m.tab != TabDeployments {
		return m, nil
	}
	return m.fetchDesk()
}

func (m Model) switchTab(tab Tab) (Model, tea.Cmd) {
	m.tab = tab
	return m.refreshDesk()
}

func (m Model) envOf(d Deployment) string {
	if inst := m.graph.Instance(instanceOf(d)); inst != nil {
		return inst.EnvID
	}
	return ""
}

func (m Model) handleDeskKey(key string) (Model, tea.Cmd, bool) {
	switch key {
	case "f":
		if m.logs.open {
			return m.closeLogs(), nil, true
		}
		d, ok := m.desk.selection()
		if !ok {
			return m.say("no deployment to follow"), nil, true
		}
		next, cmd := m.follow(TabDeployments, m.envOf(d), instanceOf(d), d)
		return next, cmd, true
	case "c":
		d, ok := m.desk.selection()
		if !ok {
			return m.say("no deployment to peek at"), nil, true
		}
		next, cmd := m.openPeek(d.ID, instanceOf(d), m.envOf(d))
		return next, cmd, true
	case "y":
		return m.askDesk(actionApprove), nil, true
	case "x":
		return m.askDesk(actionReject), nil, true
	}
	return m, nil, false
}

func (m Model) askDesk(kind actionKind) Model {
	if m.cfg.ReadOnly {
		return m.refuseReadOnly(kind)
	}
	d, ok := m.desk.selection()
	if !ok {
		return m.say("no deployment selected")
	}
	inst := instanceOf(d)
	if d.Status != string(deployments.StatusProposed) {
		return m.say(fmt.Sprintf("%s's deployment is %s; only a proposed one can be %s", inst, strings.ToLower(d.Status), strings.TrimSuffix(kind.verb(), "e")+"ed"))
	}
	return m.confirmAction(pendingAction{
		kind: kind, instanceID: inst, envID: m.envOf(d), deploymentID: d.ID,
		what: "the proposed " + strings.ToLower(d.Action), message: d.Message,
	})
}

func (m Model) openPeek(id, inst, env string) (Model, tea.Cmd) {
	m.peek = &peekView{id: id, inst: inst, env: env, loading: true}
	api, ctx := m.api, m.ctx()
	return m, func() tea.Msg {
		peek, err := api.PeekDeployment(ctx, inst, id)
		return peekMsg{id: id, peek: peek, err: err}
	}
}

func (m Model) handlePeekKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	p := *m.peek
	_, room := m.peekSize()
	switch msg.String() {
	case "ctrl+c":
		return m.closeLogs(), tea.Quit
	case "esc", "c", "q":
		m.peek = nil
		return m, nil
	case "down", "j":
		p.scroll++
	case "up", "k":
		p.scroll--
	case "pgdown", " ", "f":
		p.scroll += room
	case "pgup", "b":
		p.scroll -= room
	case "home", "g":
		p.scroll = 0
	case "end", "G":
		p.scroll = len(m.peekLines())
	}
	p.scroll = max(min(p.scroll, len(m.peekLines())-room), 0)
	m.peek = &p
	return m, nil
}

// peekSize is the width of the text in the config box and how many lines
// of it fit.
func (m Model) peekSize() (int, int) {
	return min(m.width-4, 110) - 6, max(m.bodyHeight()-4, 1)
}

func (m Model) peekLines() []string {
	p := m.peek
	w, _ := m.peekSize()
	head := []string{
		styleHeading.Render("CONFIG") + " " + styleTitle.Render(clip(p.inst, w-7)),
		styleMuted.Render(clip(envLabel(m.graph, p.env), w)),
		"",
	}
	switch {
	case p.loading:
		return append(head, styleFaint.Render("reading the deployment…"))
	case p.err != nil:
		return append(head, styleRed.Render(clip("could not read the config: "+clean(p.err.Error()), w)))
	}
	return append(head, peekBody(p.peek, m.cfg.Now(), w)...)
}

func peekBody(p DeploymentPeek, now time.Time, w int) []string {
	d := p.Deployment
	meta := []string{strings.ToLower(clean(d.Action)), strings.ToLower(clean(d.Status))}
	if d.DeployedBy != "" {
		meta = append(meta, "by "+clean(d.DeployedBy))
	}
	if !d.CreatedAt.IsZero() {
		meta = append(meta, ago(now, d.CreatedAt))
	}
	version := "bundle " + clean(cmp.Or(p.Version.Target, d.Version))
	if p.Baseline != nil && !p.Version.Equal && p.Version.Source != "" {
		version = "bundle " + clean(p.Version.Source) + " → " + clean(p.Version.Target)
	}
	out := []string{styleMuted.Render(clip(strings.Join(meta, " · "), w)), clip(version, w), "", styleHeading.Render("MESSAGE")}
	if msg := strings.TrimSpace(cleanLines(d.Message)); msg != "" {
		for _, line := range strings.Split(msg, "\n") {
			out = append(out, strings.Split(indent(wrap(line, w-2), "  "), "\n")...)
		}
	} else {
		out = append(out, styleFaint.Render("  no message"))
	}
	out = append(out, "", styleHeading.Render("PARAMS"))
	if p.Baseline != nil {
		out = append(out, styleMuted.Render(clip("against the last completed provision, "+ago(now, createdAt(*p.Baseline)), w)))
	} else {
		out = append(out, styleMuted.Render(clip("no completed provision before it, so every param is new", w)))
	}
	return append(out, paramLines(p.Params, w)...)
}

func ago(now, t time.Time) string {
	if a := Ago(now, t); a != "just now" {
		return a + " ago"
	}
	return "just now"
}

func paramLines(params []types.ParamComparison, w int) []string {
	var changed []types.ParamComparison
	for _, c := range params {
		if !c.Equal {
			changed = append(changed, c)
		}
	}
	if len(changed) == 0 {
		return []string{styleFaint.Render("  no param changes")}
	}
	slices.SortFunc(changed, func(a, b types.ParamComparison) int { return strings.Compare(a.Path, b.Path) })
	pathW := 0
	for _, c := range changed {
		pathW = max(pathW, lipgloss.Width(clean(c.Path)))
	}
	pathW = min(pathW, w/2)
	out := make([]string, 0, len(changed)+1)
	for _, c := range changed {
		out = append(out, paramLine(c, pathW, w))
	}
	if same := len(params) - len(changed); same > 0 {
		out = append(out, styleFaint.Render(fmt.Sprintf("  %s unchanged", plural(same, "param"))))
	}
	return out
}

func paramLine(c types.ParamComparison, pathW, w int) string {
	path := clean(c.Path)
	mark, style, text := "~", styleDeploy, shown(path, c.Source)+" → "+shown(path, c.Target)
	switch {
	case !c.Source.Present:
		mark, style, text = "+", styleTeal, shown(path, c.Target)
	case !c.Target.Present:
		mark, style, text = "-", styleRed, shown(path, c.Source)
	}
	return style.Render(mark) + " " + pad(clip(path, pathW), pathW) + "  " + clip(text, w-pathW-4)
}

var (
	secretPath  = regexp.MustCompile(`(?i)(secret|passw|token|credential|private|cert|auth|api_?key|access_?key|signing|key$)`)
	secretValue = regexp.MustCompile(`-----BEGIN|^(AKIA|ASIA)[0-9A-Z]{16}$|^eyJ[\w-]+\.[\w-]+\.|^gh[pousr]_|^xox[abpr]-|^sk-`)
)

const masked = "••••••"

func shown(path string, v types.ParamValue) string {
	switch {
	case !v.Present:
		return "unset"
	case secret(path, v.Value):
		return masked
	case v.Value == "":
		return `""`
	}
	return clean(v.Value)
}

func secret(path, value string) bool {
	return secretPath.MatchString(path) || secretValue.MatchString(value)
}

func (m Model) peekBox(h int) string {
	w, room := m.peekSize()
	lines := m.peekLines()
	scroll := max(min(m.peek.scroll, len(lines)-room), 0)
	visible := lines[scroll:min(scroll+room, len(lines))]
	if scroll+room < len(lines) {
		visible[len(visible)-1] = styleFaint.Render(fmt.Sprintf("↓ %d more", len(lines)-scroll-room+1))
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorPanel).Padding(1, 2).
		Width(w + 4).Render(strings.Join(visible, "\n"))
	return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) deploymentsBody(h int) string {
	if m.logs.open && m.logs.tab == TabDeployments {
		return m.logsPanel(m.width, h)
	}
	if !m.desk.read {
		msg := styleMuted.Render("reading the deployments in view…")
		if m.desk.err != nil {
			msg = styleRed.Render("could not read deployments: " + clean(m.desk.err.Error()))
		}
		return lipgloss.Place(m.width, h, lipgloss.Center, lipgloss.Center, msg)
	}
	var lines []string
	if m.desk.err != nil {
		lines = append(lines, styleRed.Render(clip("could not refresh deployments: "+clean(m.desk.err.Error()), m.width)))
	}
	b, w, now := m.desk.board, m.width-2, m.cfg.Now()
	at, selLine := 0, 0
	section := func(title string, rows []Deployment, empty string, row func(Deployment) string, style lipgloss.Style) {
		lines = append(lines, title)
		if len(rows) == 0 {
			lines = append(lines, styleFaint.Render("  "+empty))
		}
		for _, d := range rows {
			text := clip(row(d), w)
			if at == m.desk.idx {
				selLine = len(lines)
				text = styleSelected.Render(pad(text, w))
			} else {
				text = style.Render(text)
			}
			lines, at = append(lines, text), at+1
		}
		lines = append(lines, "")
	}
	section(styleHeading.Render(fmt.Sprintf("AWAITING APPROVAL (%d)", len(b.Awaiting))), b.Awaiting, "nothing awaits approval",
		func(d Deployment) string { return m.awaitingRow(d, now) }, lipgloss.NewStyle().Foreground(colorOrange))
	section(styleHeading.Render(fmt.Sprintf("IN FLIGHT (%d)", len(b.InFlight))), b.InFlight, "nothing is deploying",
		func(d Deployment) string { return m.inFlightRow(d, now) }, styleDeploy.UnsetBold())
	section(styleMuted.Render("RECENT"), b.Recent, "no finished deployments in view",
		func(d Deployment) string { return m.recentRow(d, now) }, styleFaint)
	start := max(selLine-h+2, 0)
	return strings.Join(lines[start:min(start+h, len(lines))], "\n")
}

func (m Model) awaitingRow(d Deployment, now time.Time) string {
	by := cmp.Or(d.DeployedBy, "someone")
	return fmt.Sprintf("⏸ %-30s %-26s %-12s %-20s %-8s %s", clip(instanceOf(d), 30), clip(envLabel(m.graph, m.envOf(d)), 26),
		strings.ToLower(d.Action), clip(by, 20), ago(now, createdAt(d)), firstLine(strings.TrimSpace(d.Message)))
}

func (m Model) inFlightRow(d Deployment, now time.Time) string {
	elapsed := ""
	if at := createdAt(d); !at.IsZero() {
		elapsed = clock(now.Sub(at))
	}
	return fmt.Sprintf("⟳ %-30s %-26s %-12s %-10s %s", clip(instanceOf(d), 30), clip(envLabel(m.graph, m.envOf(d)), 26),
		strings.ToLower(d.Action), strings.ToLower(d.Status), elapsed)
}

func (m Model) recentRow(d Deployment, now time.Time) string {
	mark, took := "✓", ""
	if d.Status != string(deployments.StatusCompleted) {
		mark = "✕"
	}
	if d.ElapsedTime > 0 {
		took = clock(time.Duration(d.ElapsedTime) * time.Second)
	}
	return fmt.Sprintf("%s %-30s %-26s %-12s %-10s %7s  %-8s %s", mark, clip(instanceOf(d), 30), clip(envLabel(m.graph, m.envOf(d)), 26),
		strings.ToLower(d.Action), strings.ToLower(d.Status), took, ago(now, createdAt(d)), d.DeployedBy)
}
