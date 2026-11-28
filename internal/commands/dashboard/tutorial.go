package dashboard

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/stash"
)

// Tutorial returns a made-up organization that shows everything the map
// draws at once, and cfg set up to walk through it with nothing read from or
// sent to Massdriver.
func Tutorial(cfg Config) (API, Config) {
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}
	rec := demoRecording(now)
	cfg.Org, cfg.Focus, cfg.Depth = rec.Org, rec.Focus, demoDepth
	cfg.Refresh, cfg.StashPoll = 0, 0
	if cfg.EventPoll <= 0 {
		cfg.EventPoll = demoEventPoll
	}
	cfg.Watch, cfg.Links, cfg.Replay = false, nil, 0
	cfg.Theme = ThemeConstellation
	// A frozen clock keeps the made-up alarms, deploys, and traffic as fresh
	// as when the walkthrough opened, however long it stays open.
	cfg.Now = func() time.Time { return now }
	cfg.Tutorial = true
	return tutorialAPI{recordedAPI{rec: rec}}, cfg
}

const (
	demoDepth     = 2
	demoEventPoll = 5 * time.Second
	// tutorialFlashEvery leaves a beat between one flash fading and the next
	// rising.
	tutorialFlashEvery = FlashFrames + 8
	tutorialBoxWidth   = 66
)

type tutorialAPI struct{ recordedAPI }

func (a tutorialAPI) Events(ctx context.Context, since time.Time) (Log, error) {
	return a.History(ctx, since)
}

func (a tutorialAPI) DeploymentLogs(_ context.Context, deploymentID string) (DeploymentLog, error) {
	for _, env := range a.rec.Environments {
		for _, inst := range env.Instances {
			if d := inst.LatestDeployment; d != nil && d.ID == deploymentID {
				return DeploymentLog{Status: d.Status, Batches: demoLogs(a.rec.End)}, nil
			}
		}
	}
	return DeploymentLog{}, errNotRecorded
}

type tutorialState struct {
	on   bool
	step int
	// shown is the step the view was last set up for; -1 before the first.
	shown int
	at    int
	fired int
	fires int
}

type tutorialStep struct {
	title string
	text  string
	// env is selected and locked on; empty leaves the selection alone.
	env    string
	blast  bool
	events []Event
	marks  func() []string
}

func (m Model) handleTutorialKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	last := len(tutorialSteps) - 1
	switch msg.String() {
	case "right", "l", " ", "enter":
		if m.tutorial.step == last {
			m.tutorial.on, m.blastOn = false, true
			m.refocusSeq++
			return m, nil
		}
		m.tutorial.step++
	case "left", "h":
		m.tutorial.step = max(m.tutorial.step-1, 0)
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	default:
		return m, nil
	}
	return m.showStep(), nil
}

func (m Model) showStep() Model {
	if m.graph == nil {
		return m
	}
	st := tutorialSteps[m.tutorial.step]
	m.tab, m.help, m.picked, m.filter = TabConstellation, false, "", ""
	m.blastOn = st.blast
	if st.env != "" && m.graph.Env(st.env) != nil {
		m.selected = st.env
	}
	m.tutorial.shown, m.tutorial.at = m.tutorial.step, m.frame
	if len(st.events) > 0 {
		m = m.fireTutorialEvent()
	}
	return m
}

func (m Model) tickTutorial() Model {
	if !m.tutorial.on || m.graph == nil {
		return m
	}
	if m.tutorial.shown != m.tutorial.step {
		return m.showStep()
	}
	if len(tutorialSteps[m.tutorial.step].events) > 0 && m.frame-m.tutorial.fired >= tutorialFlashEvery {
		return m.fireTutorialEvent()
	}
	return m
}

func (m Model) fireTutorialEvent() Model {
	events := tutorialSteps[m.tutorial.step].events
	ev := events[m.tutorial.fires%len(events)]
	m.tutorial.fires++
	m.tutorial.fired = m.frame
	ev.ID, ev.At = fmt.Sprintf("tutorial-%d", m.tutorial.fires), m.cfg.Now()
	m.flashes = maps.Clone(m.flashes)
	if m.flashes == nil {
		m.flashes = map[string]Flash{}
	}
	m.flashes[ev.EnvID] = Flash{Label: EventLabel(ev), Emoji: EventEmoji(ev.Type)}
	m.events = append([]Event{ev}, m.events...)
	m.events = m.events[:min(len(m.events), eventsKept)]
	return m
}

func (m Model) tutorialLockOn() (string, int, bool) {
	if !m.tutorial.on || m.tutorial.shown != m.tutorial.step {
		return "", 0, false
	}
	env, age := tutorialSteps[m.tutorial.step].env, m.frame-m.tutorial.at
	if env == "" || age >= lockOnFrames {
		return "", 0, false
	}
	return env, age, true
}

// tutorialOverlay lays the step's caption over the map, at the bottom unless
// what it explains sits there.
func (m Model) tutorialOverlay(body string) string {
	canvasW, h := m.canvasSize()
	w := min(canvasW-2, tutorialBoxWidth)
	if w < 24 {
		return body
	}
	box := strings.Split(m.tutorialCaption(w), "\n")
	lines := strings.Split(body, "\n")
	top := len(lines) - len(box)
	if p, ok := m.animatedPos()[m.selected]; ok && int(p.Y)/4 > h/2 {
		top = 1
	}
	for i, line := range box {
		if row := top + i; row >= 0 && row < len(lines) {
			lines[row] = " " + line + ansi.TruncateLeft(lines[row], lipgloss.Width(line)+1, "")
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) tutorialCaption(w int) string {
	st := tutorialSteps[m.tutorial.step]
	inner := w - 4
	rows := []string{styleTitle.Render(st.title)}
	if st.text != "" {
		rows = append(rows, wrap(st.text, inner))
	}
	if st.marks != nil {
		if st.text != "" {
			rows = append(rows, "")
		}
		for _, mark := range st.marks() {
			rows = append(rows, lipgloss.NewStyle().MaxWidth(inner).Render(mark))
		}
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorTeal).
		Padding(0, 1).Width(w - 2).Render(strings.Join(rows, "\n"))
}

func (m Model) tutorialFooter() string {
	sep := styleFaint.Render("  ·  ")
	next := "next"
	if m.tutorial.step == len(tutorialSteps)-1 {
		next = "explore"
	}
	line := styleKey.Render(fmt.Sprintf("%d/%d", m.tutorial.step+1, len(tutorialSteps))) + sep +
		styleKey.Render("→") + styleMuted.Render(" "+next) + sep +
		styleKey.Render("←") + styleMuted.Render(" back") + sep +
		styleKey.Render("q") + styleMuted.Render(" quit")
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}

const (
	demoNetwork = "network-prod"
	demoStore   = "ecomm-prod"
	demoStaging = "ecomm-staging"
	demoData    = "data-prod"
	demoML      = "data-ml"
	demoLake    = "data-lake"
	demoVault   = "secops-vault"
)

var demoFlashes = []Event{
	{Type: "deployment.started", EnvID: demoML, Detail: "provision", Actor: "mia"},
	{Type: "deployment.completed", EnvID: demoNetwork, Detail: "provision", Actor: "kai"},
	{Type: "deployment.failed", EnvID: demoData, Detail: "provision"},
	{Type: "alarm.fired", EnvID: demoStore, Detail: "CPU above 90%"},
	{Type: "stash.set", EnvID: demoStaging, Detail: "release = v42"},
}

var tutorialSteps = []tutorialStep{
	{
		title: "The constellation", env: demoStore,
		text: "Each ring is an environment, with its name and project beneath it. Lines join environments that share " +
			"resources. The map centers on one environment, here Storefront / prod, and reaches a few hops out " +
			"from it: --env picks the center and --depth how far.",
	},
	{
		title: "Rings", env: demoData,
		text: "A ring shows the environment's instances by status, each status taking its share of the circle. " +
			"Data Platform / prod has all four:",
		marks: func() []string {
			return []string{
				dot(colorGreen) + " provisioned   " + dot(colorRed) + " failed",
				dot(colorBlue) + " not deployed  " + dot(colorGray) + " decommissioned",
			}
		},
	},
	{
		title: "Instances", env: demoStore,
		text: "The dots around a ring are its instances, each in its status color. Applications are the larger dots; " +
			"the side panel marks them ◆ with their runtime. An instance that reports md/v1/health takes its " +
			"health's color instead, like Storefront / prod's degraded cache, and one with an alarm firing blinks red.",
		marks: func() []string {
			return []string{dot(colorGreen) + " healthy  " + dot(colorOrange) + " degraded  " + dot(colorRed) + " down"}
		},
	},
	{
		title: "Alarms", env: demoStore,
		text: "A firing alarm puts a red core in the ring and sends a red pulse out from it. ▲1 at the ring's upper " +
			"right counts the alarms firing there, and the header counts the instances alarming. The Alarms tab (3) " +
			"lists every one in view.",
	},
	{
		title: "Deploying", env: demoStaging,
		text: "A comet circling a ring is a deployment running there, and the deploying instance's dot blinks. " +
			"The header counts deployments in flight, and the side panel names the instance, the action, and how " +
			"long it has run. f follows its logs.",
	},
	{
		title: "Selection", env: demoNetwork,
		text: "The spinning halo marks the selected environment: its name is highlighted and the side panel " +
			"describes it. The arrow keys move the selection, and once it rests the map re-centers " +
			"there. ⏸ in the side panel is a deployment awaiting approval; n picks its instance and y approves it.",
	},
	{
		title: "Sparks", env: demoData,
		text: "Sparks run along each line from the environment that produces a resource to the one that uses it. " +
			"Where the producing instance sets md/v1/traffic, their number and pace follow its requests per second, " +
			"the line is labeled with the rate and p95 latency, and errors in the last five minutes show as ✕N with " +
			"amber sparks, red at ten or more. Without it, a line that shares more resources carries more sparks.",
	},
	{
		title: "Forks", env: demoStaging,
		text: "A dashed line joins an environment to the one it was forked from: Storefront / staging was forked " +
			"from Storefront / prod. The side panel says so under its name.",
	},
	{
		title: "Blast radius", env: demoStore, blast: true,
		text: "The blast radius is everything downstream that uses the selected environment's resources, directly or " +
			"through others. With an alarm firing, it starts at the alarmed instance. Halos, lines, and instance dots " +
			"light up by hops from the source, and the side panel lists them. b turns it off and on.",
		marks: func() []string {
			return []string{dot(hopColor(0)) + " source  " + dot(hopColor(1)) + " one hop  " + dot(hopColor(2)) + " two or more"}
		},
	},
	{
		title: "Upgrades", env: demoStore,
		text: "↑1 at a ring's upper left counts the instances that can move to a newer bundle version their " +
			"constraint allows. The side panel shows the version each would move to, and the Upgrades tab (4) " +
			"lists every one in view.",
	},
	{
		title: "Banners", env: demoStaging,
		text: "⚑ beside a name flags a banner, set in the environment's md/v1/banner. The selected environment's " +
			"banner runs across the top of the map, or else the most urgent in view, colored by its level:",
		marks: func() []string {
			return []string{dot(levelColor("info")) + " info  " + dot(levelColor("warning")) + " warning  " + dot(levelColor("error")) + " error"}
		},
	},
	{
		title: "Notes and links", env: demoStore,
		text: "✎ marks a note: an annotation you wrote on a project, environment, component, instance, or link, " +
			"rendered from its Liquid template with the stash. Only you see your notes. ↗ links to dashboards and runbooks come " +
			"from md/v1/links.",
	},
	{
		title: "Out of view", env: demoLake,
		text: "🔭 under a name marks an environment past the edge of the view: one in view links to it, but it is " +
			"more hops from the center than the dashboard reads. Select it and the map re-centers there.",
	},
	{
		title: "No access", env: demoVault,
		text: "A faint ring marked no access is an environment you cannot view whose resources instances in view " +
			"use. Its lines still show who depends on it. The metro and radar themes draw it as ◌.",
	},
	{
		title: "Flashes", events: demoFlashes,
		text: "Where something happens, the environment ripples and an emoji rises from it. --flashes words shows " +
			"the event's name instead, and --flashes off only the ripple.",
		marks: func() []string {
			return []string{"🚀 deploy started  ✅ deploy finished  💥 deploy failed", "🚨 alarm  📌 stash key set"}
		},
	},
	{
		title: "Ticker", events: demoFlashes,
		text: "The line under the map follows the audit log, newest first: each event's emoji, how long ago, where, " +
			"what, and who. Stash changes join it as they arrive. t replays the audit log onto the map, from five " +
			"minutes to a day back.",
	},
	{
		title: "Side panel", env: demoStore, blast: true,
		text: "The side panel describes the selected environment: its project, deploys in flight, links, banner, " +
			"and notes; its instances by status; firing alarms; the environments it uses resources from and feeds; " +
			"its blast radius; and each instance with its runtime, upgrade, and widgets. It shows on terminals at " +
			"least 90 columns wide, and ? swaps in the legend.",
	},
	{
		title: "Keys",
		marks: func() []string {
			return []string{
				keyHint("1-5", "Constellation, Runtimes, Alarms, Upgrades, Deployments"),
				keyHint("/", "find an environment anywhere in the organization"),
				keyHint("f", "follow the selected environment's deploy logs"),
				keyHint("n", "pick an instance  ") + keyHint("y", "approve its deploy  ") + keyHint("d", "deploy it"),
				keyHint("t", "replay  ") + keyHint("w", "write up an incident  ") + keyHint("e", "export SVG"),
				keyHint("v", "theme  ") + keyHint("p", "palette  ") + keyHint("?", "every key"),
			}
		},
	},
}

func demoRecording(now time.Time) *Recording {
	return &Recording{
		Org: "orbital-outfitters", Focus: demoStore, Start: now.Add(-30 * time.Minute), End: now,
		Environments: slices.Concat(demoStorefront(now), []Environment{demoNetworkEnv(now)}, demoDataEnvs()),
		Runtimes: []Runtime{
			{ID: "node", Latest: Bundle{Name: "node", Version: "22.4.0"}, Adoption: []Adoption{
				{Version: "20.11.0", ApplicationCount: 1, InstanceCount: 1}, {Version: "22.4.0", ApplicationCount: 1, InstanceCount: 1},
			}},
			{ID: "python", Latest: Bundle{Name: "python", Version: "3.12.4"}, Adoption: []Adoption{
				{Version: "3.11.6", ApplicationCount: 1, InstanceCount: 1},
			}},
		},
		Applications: []Application{
			{ID: "storefront-api", Latest: *demoApp("storefront-api", "1.5.0", "node", "22.4.0")},
			{ID: "orders-etl", Latest: *demoApp("orders-etl", "2.3.0", "python", "3.11.6")},
		},
		Annotations: map[string][]Annotation{"ecomm": {
			{ID: "note-oncall", Kind: "NOTE", Surfaces: []string{SurfaceCanvasLegend}, Template: "On call: {{ environment.stash.oncall }}",
				Entity: AnnotationEntity{Typename: "Environment", ID: demoStore}},
			{ID: "note-failover", Kind: "NOTE", Surfaces: []string{SurfaceCanvasNode}, Template: "Failover to the replica is manual.",
				Entity: AnnotationEntity{Typename: "Instance", ID: "ecomm-prod-database"}},
		}},
		Stash:  demoStash(now),
		Events: demoEvents(now),
	}
}

func demoNetworkEnv(now time.Time) Environment {
	return Environment{ID: demoNetwork, Name: "prod", Project: &Ref{ID: "network", Name: "Network"}, Instances: []Instance{
		{ID: "network-prod-vpc", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-vpc", Version: "3.2.0"}, Cost: demoCost(310),
			Resources: []InstanceResource{
				demoUsedBy("ecomm-prod-api", demoStore), demoUsedBy("ecomm-staging-api", demoStaging), demoUsedBy("data-prod-stream", demoData),
			}},
		{ID: "network-prod-dns", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-route53", Version: "1.1.0"},
			LatestDeployment: &Deployment{ID: "dep-dns", Status: "PROPOSED", Action: "PROVISION", CreatedAt: stamp(now, 3*time.Minute),
				Version: "1.1.0", Message: "Lower the TTL to 60s", DeployedBy: "kai"}},
	}}
}

func demoStorefront(now time.Time) []Environment {
	alarm := AlarmsPage{Items: []Alarm{{DisplayName: "CPU above 90%", CurrentState: &AlarmState{
		Status: "ALARM", Message: "CPUUtilization was 97% for 10 minutes", OccurredAt: stamp(now, 7*time.Minute),
	}}}}
	prod := Environment{ID: demoStore, Name: "prod", Project: &Ref{ID: "ecomm", Name: "Storefront"}, Instances: []Instance{
		{ID: "ecomm-prod-database", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-rds-postgres", Version: "4.0.1"},
			FiringAlarmCount: 1, Alarms: alarm, Cost: demoCost(1840),
			Dependencies: []Dependency{demoDependsOn("network-prod-vpc", demoNetwork)},
			Resources:    []InstanceResource{demoUsedBy("data-prod-stream", demoData)}},
		{ID: "ecomm-prod-api", Status: StatusProvisioned, Bundle: demoApp("storefront-api", "1.4.2", "node", "20.11.0"),
			DeployedVersion: "1.4.2", AvailableUpgrade: "1.5.0", Cost: demoCost(620),
			Dependencies: []Dependency{
				demoDependsOn("network-prod-vpc", demoNetwork), demoDependsOn("ecomm-prod-database", demoStore), demoDependsOn("secops-vault-secrets", demoVault),
			}},
		{ID: "ecomm-prod-cache", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-elasticache-redis", Version: "2.0.0"}, Cost: demoCost(240),
			Dependencies: []Dependency{demoDependsOn("network-prod-vpc", demoNetwork)}},
	}}
	staging := Environment{ID: demoStaging, Name: "staging", Project: &Ref{ID: "ecomm", Name: "Storefront"}, Parent: &Ref{ID: demoStore}, Instances: []Instance{
		{ID: "ecomm-staging-database", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-rds-postgres", Version: "4.0.1"}, Cost: demoCost(410)},
		{ID: "ecomm-staging-api", Status: StatusProvisioned, Bundle: demoApp("storefront-api", "1.5.0", "node", "22.4.0"), Cost: demoCost(150),
			DeployedVersion: "1.5.0",
			LatestDeployment: &Deployment{ID: "dep-staging-api", Status: "RUNNING", Action: "PROVISION", CreatedAt: stamp(now, 2*time.Minute),
				Version: "1.5.0", Message: "Move to Node 22", DeployedBy: "jules"},
			Dependencies: []Dependency{demoDependsOn("network-prod-vpc", demoNetwork), demoDependsOn("ecomm-staging-database", demoStaging)}},
	}}
	return []Environment{prod, staging}
}

func demoDataEnvs() []Environment {
	project := &Ref{ID: "data", Name: "Data Platform"}
	stream := []Dependency{demoDependsOn("data-prod-stream", demoData)}
	prod := Environment{ID: demoData, Name: "prod", Project: project, Instances: []Instance{
		{ID: "data-prod-stream", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-kinesis", Version: "1.3.0"}, Cost: demoCost(530),
			Dependencies: []Dependency{demoDependsOn("ecomm-prod-database", demoStore), demoDependsOn("network-prod-vpc", demoNetwork)},
			Resources:    []InstanceResource{demoUsedBy("data-ml-features", demoML)}},
		{ID: "data-prod-warehouse", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-redshift", Version: "2.1.0"}, Cost: demoCost(2200), Dependencies: stream},
		{ID: "data-prod-etl", Status: StatusFailed, Bundle: demoApp("orders-etl", "2.3.0", "python", "3.11.6"), Dependencies: stream},
		{ID: "data-prod-backfill", Status: StatusInitialized, Bundle: &Bundle{Name: "aws-batch", Version: "0.9.0"}, Dependencies: stream},
		{ID: "data-prod-archive", Status: StatusDecommissioned, Bundle: &Bundle{Name: "aws-s3", Version: "1.0.0"}},
	}}
	ml := Environment{ID: demoML, Name: "ml", Project: project, Instances: []Instance{
		{ID: "data-ml-features", Status: StatusProvisioned, Bundle: &Bundle{Name: "feature-store", Version: "0.4.0"}, Cost: demoCost(380),
			Dependencies: []Dependency{demoDependsOn("data-prod-stream", demoData), demoDependsOn("data-lake-bucket", demoLake)}},
		{ID: "data-ml-training", Status: StatusProvisioned, Bundle: &Bundle{Name: "sagemaker", Version: "1.2.0"}, Cost: demoCost(960),
			Dependencies: []Dependency{demoDependsOn("data-ml-features", demoML)}},
	}}
	lake := Environment{ID: demoLake, Name: "lake", Project: project, Instances: []Instance{
		{ID: "data-lake-bucket", Status: StatusProvisioned, Bundle: &Bundle{Name: "aws-s3", Version: "1.0.0"}, Cost: demoCost(120),
			Resources: []InstanceResource{demoUsedBy("data-ml-features", demoML)}},
	}}
	return []Environment{prod, ml, lake}
}

func demoStash(now time.Time) []StashEntry {
	instance := func(id string) Scope { return Scope{Kind: stash.ScopeInstance, ID: id} }
	env := func(id string) Scope { return Scope{Kind: stash.ScopeEnvironment, ID: id} }
	jsonEntry := func(scope Scope, key string, data map[string]any) StashEntry {
		return StashEntry{Scope: scope, Key: key, Format: string(stash.FormatJSON), Data: data, UpdatedAt: now.Add(-time.Minute)}
	}
	text := func(scope Scope, key, value string) StashEntry {
		return StashEntry{Scope: scope, Key: key, Format: string(stash.FormatText), Value: &value, UpdatedAt: now.Add(-time.Minute)}
	}
	return []StashEntry{
		jsonEntry(instance("ecomm-prod-database"), stash.KeyTrafficV1, map[string]any{"rps": 850.0, "errors_5m": 0.0, "p95_ms": 12.0}),
		jsonEntry(instance("ecomm-prod-database"), stash.KeyUsageV1, map[string]any{"value": 412.0, "max": 500.0, "label": "GB"}),
		jsonEntry(instance("data-prod-stream"), stash.KeyTrafficV1, map[string]any{"rps": 3400.0, "errors_5m": 14.0, "p95_ms": 48.0}),
		text(instance("ecomm-prod-cache"), stash.KeyHealthV1, HealthDegraded),
		text(instance("ecomm-prod-api"), stash.KeyHealthV1, HealthHealthy),
		jsonEntry(env(demoStaging), stash.KeyBannerV1, map[string]any{"message": "Load test until 17:00 UTC; expect errors", "level": "warning"}),
		jsonEntry(env(demoStore), stash.KeyLinksV1, map[string]any{"links": []any{
			map[string]any{"label": "Runbook", "url": "https://runbooks.example.com/storefront"},
			map[string]any{"label": "Grafana", "url": "https://grafana.example.com/d/storefront"},
		}}),
		text(env(demoStore), "oncall", "#storefront-oncall"),
	}
}

func demoEvents(now time.Time) []Event {
	ev := func(id string, ago time.Duration, kind, env, detail, actor string) Event {
		return Event{ID: id, At: now.Add(-ago), Type: kind, EnvID: env, Detail: detail, Actor: actor}
	}
	return []Event{
		ev("demo-1", 28*time.Minute, "deployment.completed", demoML, "provision", "mia"),
		ev("demo-2", 21*time.Minute, "stash.set", demoStaging, "md/v1/banner = Load test until 17:00 UTC", "jules"),
		ev("demo-3", 14*time.Minute, "deployment.failed", demoData, "provision", ""),
		ev("demo-4", 7*time.Minute, "alarm.fired", demoStore, "CPU above 90%", ""),
		ev("demo-5", 3*time.Minute, "deployment.proposed", demoNetwork, "provision", "kai"),
		ev("demo-6", 2*time.Minute, "deployment.started", demoStaging, "provision", "jules"),
	}
}

func demoLogs(now time.Time) []LogBatch {
	start := now.Add(-2 * time.Minute)
	lines := []string{
		"Initializing the backend...",
		"Refreshing state... [id=storefront-api]",
		"aws_ecs_task_definition.api: Creating...",
		"aws_ecs_task_definition.api: Creation complete after 2s [id=storefront-api:43]",
		"aws_ecs_service.api: Modifying... [id=storefront-api]",
		"aws_ecs_service.api: Still modifying... [00m40s elapsed]",
	}
	out := make([]LogBatch, 0, len(lines))
	for i, line := range lines {
		out = append(out, LogBatch{Timestamp: start.Add(time.Duration(i) * 10 * time.Second), Message: line})
	}
	return out
}

func demoApp(name, version, runtime, runtimeVersion string) *Bundle {
	return &Bundle{Name: name, Version: version, Runtime: &Bundle{Name: runtime, Version: runtimeVersion}}
}

func demoCost(amount float64) Cost {
	return Cost{LastMonth: CostSample{Amount: &amount, Currency: "USD"}}
}

func demoDependsOn(instanceID, envID string) Dependency {
	return Dependency{Field: "upstream", Resource: Resource{
		ID: instanceID + ".out", Origin: "PROVISIONED", Instance: &ResourceInstance{ID: instanceID, Environment: &Ref{ID: envID}},
	}}
}

func demoUsedBy(instanceID, envID string) InstanceResource {
	return InstanceResource{Resource: ProducedResource{RemoteReferences: RemoteReferences{
		Items: []RemoteReference{{Instance: &ResourceInstance{ID: instanceID, Environment: &Ref{ID: envID}}}},
	}}}
}

func stamp(now time.Time, ago time.Duration) string {
	return now.Add(-ago).UTC().Format(time.RFC3339)
}
