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
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// Deployer is the SDK surface the dashboard changes things through.
type Deployer interface {
	GetInstance(ctx context.Context, id string) (*types.Instance, error)
	CreateDeployment(ctx context.Context, instanceID string, in deployments.CreateInput) (*types.Deployment, error)
	ApproveDeployment(ctx context.Context, id string) (*types.Deployment, error)
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
)

func (k actionKind) verb() string {
	if k == actionApprove {
		return "approve"
	}
	return "deploy"
}

type pendingAction struct {
	kind         actionKind
	instanceID   string
	envID        string
	deploymentID string
	what         string
	envs, insts  int
}

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
	if env == nil || len(env.Instances) == 0 {
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
		return m.say("the dashboard is read-only; restart it without --read-only to " + kind.verb())
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
	}
	act.envs, act.insts = m.graph.BlastRadius(inst.ID).Downstream()
	m.confirm = &act
	return m
}

func (m Model) say(toast string) Model {
	m.toast, m.toastAt = toast, m.frame
	return m
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	act := *m.confirm
	m.confirm = nil
	switch msg.String() {
	case "y", "enter":
	default:
		return m.say(act.verb() + " cancelled"), nil
	}
	api, ctx := m.api, m.ctx()
	return m, func() tea.Msg {
		var err error
		if act.kind == actionApprove {
			err = api.ApproveDeployment(ctx, act.deploymentID)
		} else {
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
	title := "APPROVE DEPLOYMENT"
	if act.kind == actionDeploy {
		title = "DEPLOY"
	}
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
		"",
		verb + " " + act.what + "?",
		reach,
		"",
		keyHint("y", act.verb()) + "  " + keyHint("enter", act.verb()) + "  " + styleMuted.Render("any other key cancels"),
	}
	return m.helpBox(h, strings.Join(rows, "\n"))
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
