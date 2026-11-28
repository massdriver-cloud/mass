package dashboard

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/deployments"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// Deployer is the SDK surface the dashboard changes things through.
type Deployer interface {
	GetInstance(ctx context.Context, id string) (*types.Instance, error)
	CreateDeployment(ctx context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error)
	ApproveDeployment(ctx context.Context, id string) (*types.Deployment, error)
	RejectDeployment(ctx context.Context, id string) (*types.Deployment, error)
	GetDeployment(ctx context.Context, id string) (*types.Deployment, error)
	CompareDeployments(ctx context.Context, sourceID, targetID string) (*types.DeploymentComparison, error)
	SetStash(ctx context.Context, scope stash.Scope, key string, in stash.SetInput) (*stash.Entry, error)
}

// NewDeployer returns the production [Deployer] backed by the SDK client.
func NewDeployer(c *massdriver.Client) Deployer { return sdkDeployer{c: c} }

type sdkDeployer struct{ c *massdriver.Client }

func (s sdkDeployer) GetInstance(ctx context.Context, id string) (*types.Instance, error) {
	return s.c.Instances.Get(ctx, id)
}

func (s sdkDeployer) CreateDeployment(ctx context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error) {
	return s.c.Deployments.Create(ctx, instanceID, in)
}

func (s sdkDeployer) ApproveDeployment(ctx context.Context, id string) (*types.Deployment, error) {
	return s.c.Deployments.Approve(ctx, id)
}

func (s sdkDeployer) RejectDeployment(ctx context.Context, id string) (*types.Deployment, error) {
	return s.c.Deployments.Reject(ctx, id)
}

func (s sdkDeployer) GetDeployment(ctx context.Context, id string) (*types.Deployment, error) {
	return s.c.Deployments.Get(ctx, id)
}

func (s sdkDeployer) CompareDeployments(ctx context.Context, sourceID, targetID string) (*types.DeploymentComparison, error) {
	return s.c.Deployments.Compare(ctx, sourceID, targetID)
}

func (s sdkDeployer) SetStash(ctx context.Context, scope stash.Scope, key string, in stash.SetInput) (*stash.Entry, error) {
	return s.c.Stash.Set(ctx, scope, key, in)
}

// ErrNoDeployer means the dashboard was built without a way to deploy.
var ErrNoDeployer = errors.New("this dashboard cannot deploy")

func (a *gqlAPI) ApproveDeployment(ctx context.Context, deploymentID string) error {
	if a.deployer == nil {
		return ErrNoDeployer
	}
	_, err := a.deployer.ApproveDeployment(ctx, deploymentID)
	return err
}

// Deploy provisions the instance again with the params it has now. The
// server has no "reuse saved params" mode, so they are read and sent back.
func (a *gqlAPI) Deploy(ctx context.Context, instanceID string) error {
	if a.deployer == nil {
		return ErrNoDeployer
	}
	inst, err := a.deployer.GetInstance(ctx, instanceID)
	if err != nil {
		return err
	}
	params := inst.Params
	if params == nil {
		params = map[string]any{}
	}
	_, err = a.deployer.CreateDeployment(ctx, inst.ID, deployments.CreateInput{
		Action:  deployments.ActionProvision,
		Params:  params,
		Message: "Deployed from mass dashboard",
	})
	return err
}

type actionKind int

const (
	actionApprove actionKind = iota
	actionDeploy
	actionReject
)

func (k actionKind) verb() string {
	switch k {
	case actionApprove:
		return "approve"
	case actionReject:
		return "reject"
	case actionDeploy:
	}
	return "deploy"
}

type pendingAction struct {
	kind         actionKind
	instanceID   string
	envID        string
	deploymentID string
	what         string
	message      string
	reason       string
	envs, insts  int
}

// maxReason keeps a rejection reason to what reads as one line.
const maxReason = 200

type actionMsg struct {
	action pendingAction
	err    error
}

func (m Model) pickedInstance() *InstNode {
	if m.graph == nil || m.picked == "" {
		return nil
	}
	env := m.graph.Env(m.selected)
	if env == nil || env.Restricted || env.Unloaded {
		return nil
	}
	for _, inst := range env.Instances {
		if inst.ID == m.picked {
			return inst
		}
	}
	return nil
}

func (m Model) pickInstance(key string) Model {
	if m.graph == nil {
		return m
	}
	env := m.graph.Env(m.selected)
	if env == nil || env.Restricted || env.Unloaded || len(env.Instances) == 0 {
		return m
	}
	idx := slices.IndexFunc(env.Instances, func(i *InstNode) bool { return i.ID == m.picked })
	n := len(env.Instances)
	switch {
	case idx < 0 && key == "N":
		idx = n - 1
	case idx < 0:
		idx = 0
	case key == "N":
		idx = (idx - 1 + n) % n
	default:
		idx = (idx + 1) % n
	}
	m.picked = env.Instances[idx].ID
	return m
}

func (m Model) ask(kind actionKind) Model {
	if m.cfg.ReadOnly {
		return m.refuseReadOnly(kind)
	}
	inst := m.pickedInstance()
	if inst == nil {
		return m.say("pick an instance with n first")
	}
	act := pendingAction{kind: kind, instanceID: inst.ID, envID: inst.EnvID, what: "a new deployment of its current configuration"}
	if kind == actionApprove {
		if !inst.AwaitingApproval() {
			return m.say(inst.ID + " has no deployment awaiting approval")
		}
		if inst.Deploy.ID == "" {
			return m.say("the deployment awaiting approval on " + inst.ID + " has no id; press r and try again")
		}
		act.deploymentID = inst.Deploy.ID
		act.what = "the proposed " + strings.ToLower(inst.Deploy.Action)
	} else if inst.Deploying() {
		return m.say(inst.ID + " is already deploying")
	}
	return m.confirmAction(act)
}

func (m Model) refuseReadOnly(kind actionKind) Model {
	return m.say("the dashboard is read-only; restart it without --read-only to " + kind.verb())
}

func (m Model) confirmAction(act pendingAction) Model {
	act.envs, act.insts = m.graph.BlastRadius(act.instanceID).Downstream()
	m.confirm = &act
	return m
}

func (m Model) say(toast string) Model {
	m.toast, m.toastAt = toast, m.frame
	return m
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	act := *m.confirm
	if act.kind == actionReject {
		return m.handleReasonKey(msg)
	}
	key := msg.String()
	if key == "c" && act.kind == actionApprove {
		return m.openPeek(act.deploymentID, act.instanceID, act.envID)
	}
	m.confirm = nil
	if key != "y" && key != "enter" {
		return m.say(act.verb() + " cancelled"), nil
	}
	// The map may have refreshed while the box was open.
	if inst := m.graph.Instance(act.instanceID); act.kind == actionDeploy && inst != nil && inst.Deploying() {
		return m.say(act.instanceID + " started deploying; nothing was sent"), nil
	}
	return m, m.send(act)
}

func (m Model) handleReasonKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	act := *m.confirm
	switch msg.String() {
	case "enter":
		m.confirm = nil
		return m, m.send(act)
	case "esc", "ctrl+c":
		m.confirm = nil
		return m.say(act.verb() + " cancelled"), nil
	case "backspace":
		if r := []rune(act.reason); len(r) > 0 {
			act.reason = string(r[:len(r)-1])
		}
	default:
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			r := []rune(act.reason + clean(string(msg.Runes)))
			act.reason = string(r[:min(len(r), maxReason)])
		}
	}
	m.confirm = &act
	return m, nil
}

func (m Model) send(act pendingAction) tea.Cmd {
	api, ctx := m.api, m.ctx()
	return func() tea.Msg {
		var err error
		switch act.kind {
		case actionApprove:
			err = api.ApproveDeployment(ctx, act.deploymentID)
		case actionReject:
			err = api.RejectDeployment(ctx, act.deploymentID, strings.TrimSpace(act.reason))
		case actionDeploy:
			err = api.Deploy(ctx, act.instanceID)
		}
		return actionMsg{action: act, err: err}
	}
}

func (a pendingAction) verb() string { return a.kind.verb() }

func (m Model) applyAction(msg actionMsg) (Model, tea.Cmd) {
	act := msg.action
	switch {
	case msg.err == nil && act.kind == actionApprove:
		m = m.say("✓ approved " + act.instanceID)
	case msg.err == nil && act.kind == actionReject:
		m = m.say("✓ rejected " + act.instanceID)
	case act.kind == actionReject && errors.Is(msg.err, ErrReasonNotSaved):
		m = m.say("✓ rejected " + act.instanceID + ", but " + clean(msg.err.Error()))
	case msg.err == nil:
		m = m.say("⟳ deploying " + act.instanceID)
	default:
		m = m.say("✕ " + actionError(act, msg.err))
	}
	return m.fetch()
}

func actionError(act pendingAction, err error) string {
	switch {
	case errors.Is(err, gql.ErrForbidden):
		return fmt.Sprintf("you do not have permission to %s %s", act.verb(), act.instanceID)
	case errors.Is(err, gql.ErrUnauthenticated):
		return fmt.Sprintf("could not %s %s: your credentials were refused; sign in again", act.verb(), act.instanceID)
	}
	return fmt.Sprintf("could not %s %s: %s", act.verb(), act.instanceID, clean(err.Error()))
}

func (m Model) confirmBox(h int) string {
	act := m.confirm
	title := map[actionKind]string{actionApprove: "APPROVE DEPLOYMENT", actionReject: "REJECT DEPLOYMENT", actionDeploy: "DEPLOY"}[act.kind]
	verb := strings.ToUpper(act.verb()[:1]) + act.verb()[1:]
	reach := styleTeal.Render("✸") + " nothing depends on it downstream"
	if act.envs > 0 || act.insts > 0 {
		reach = lipgloss.NewStyle().Foreground(colorOrange).Render("✸") +
			fmt.Sprintf(" %s and %s depend on it downstream", plural(act.envs, "environment"), plural(act.insts, "instance"))
	}
	rows := []string{
		styleHeading.Render(title), "",
		styleTitle.Render(act.instanceID),
		styleMuted.Render(envLabel(m.graph, act.envID)),
	}
	if act.message != "" {
		rows = append(rows, styleMuted.Render("“"+clip(firstLine(strings.TrimSpace(act.message)), 80)+"”"))
	}
	rows = append(rows, "", verb+" "+act.what+"?", reach, "")
	switch act.kind {
	case actionReject:
		rows = append(rows,
			"Reason (optional): "+act.reason+styleTeal.Render("█"), "",
			keyHint("enter", "reject")+"  "+keyHint("esc", "cancel"))
	case actionApprove:
		rows = append(rows, keyHint("y", act.verb())+"  "+keyHint("enter", act.verb())+"  "+keyHint("c", "peek the config")+"  "+
			styleMuted.Render("any other key cancels"))
	case actionDeploy:
		rows = append(rows, keyHint("y", act.verb())+"  "+keyHint("enter", act.verb())+"  "+styleMuted.Render("any other key cancels"))
	}
	return m.helpBox(h, strings.Join(rows, "\n"))
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
