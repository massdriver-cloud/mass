package dashboard

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/massdriver-cloud/mass/internal/cli"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/urls"
)

// Options configure [Run].
type Options struct {
	Config
	Out *os.File
}

// Run opens the dashboard when Out is a terminal, and otherwise writes a
// plain summary of the same data.
func Run(ctx context.Context, api API, opts Options) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if !cli.IsInteractive(opts.Out) {
		return RunSummary(ctx, api, opts.Out, opts.Now())
	}
	if opts.Open == nil {
		opts.Open = OpenBrowser
	}
	opts.Context = ctx
	model := NewModel(api, opts.Config)
	final, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(opts.Out), tea.WithContext(ctx)).Run()
	if err != nil {
		return err
	}
	if m, ok := final.(Model); ok && m.graph == nil && m.err != nil {
		return m.err
	}
	return nil
}

// OpenBrowser opens url in the system's browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(context.Background(), "open", url)
	case "windows":
		cmd = exec.CommandContext(context.Background(), "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.CommandContext(context.Background(), "xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() //nolint:errcheck // the browser's exit status is not ours to report
	return nil
}

// RunSummary writes what the dashboard shows as plain text.
func RunSummary(ctx context.Context, api API, out io.Writer, now time.Time) error {
	snap, err := Fetch(ctx, api)
	if err != nil {
		return err
	}
	g := BuildGraph(snap)
	writeEnvironments(out, g)
	writeAlarms(out, g, now)
	writeRuntimes(out, g, snap)
	return nil
}

func writeEnvironments(out io.Writer, g *Graph) {
	tbl := cli.NewTable("Environment", "Project", "Instances", "Uses resources from", "Alarms").WithWriter(out)
	for _, env := range g.Envs {
		if env.Restricted {
			continue
		}
		producers := make([]string, 0, len(g.Producers(env.ID)))
		for _, l := range g.Producers(env.ID) {
			producers = append(producers, l.From)
		}
		tbl.AddRow(env.ID, env.ProjectName, statusSummary(env), strings.Join(producers, ", "), env.FiringCount())
	}
	tbl.Print()
}

func statusSummary(env *EnvNode) string {
	parts := make([]string, 0, len(Statuses))
	for _, status := range Statuses {
		if n := env.StatusCount(status); n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, statusLabel(status)))
		}
	}
	return strings.Join(parts, ", ")
}

func writeAlarms(out io.Writer, g *Graph, now time.Time) {
	if len(g.Firing) == 0 {
		return
	}
	fmt.Fprintln(out)
	tbl := cli.NewTable("Firing", "Alarm", "Instance", "Downstream").WithWriter(out)
	for _, a := range g.Firing {
		envs, insts := g.BlastRadius(a.InstanceID).Downstream()
		tbl.AddRow(Ago(now, a.Since), a.Name, a.InstanceID, fmt.Sprintf("%d environments, %d instances", envs, insts))
	}
	tbl.Print()
}

func writeRuntimes(out io.Writer, g *Graph, snap *Snapshot) {
	if len(snap.Runtimes) == 0 {
		return
	}
	fmt.Fprintln(out)
	tbl := cli.NewTable("Runtime", "Latest", "On latest", "Applications").WithWriter(out)
	for _, rt := range snap.Runtimes {
		r := BuildRollout(g, snap, rt, BumpNone)
		tbl.AddRow(rt.ID, rt.Latest.Version, fmt.Sprintf("%d of %d instances", r.OnLatest(), len(r.Instances)), len(r.Apps))
	}
	tbl.Print()
}

type appLinks struct{ helper *urls.Helper }

// NewLinks builds web app URLs with the SDK's URL helper.
func NewLinks(helper *urls.Helper) Links { return appLinks{helper: helper} }

func (l appLinks) Environment(id string) string { return l.helper.EnvironmentURL(id) }

func (l appLinks) Instance(id string) string { return l.helper.InstanceURL(id) }

func (l appLinks) Runtime(id string) string {
	return fmt.Sprintf("%s/orgs/%s/runtimes/%s", l.helper.BaseURL, l.helper.OrgID, id)
}
