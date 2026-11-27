package dashboard

import (
	"cmp"
	"slices"
	"time"
)

// Instance statuses, as the API spells them.
const (
	StatusProvisioned    = "PROVISIONED"
	StatusFailed         = "FAILED"
	StatusInitialized    = "INITIALIZED"
	StatusDecommissioned = "DECOMMISSIONED"
)

// Statuses lists the instance statuses in the order the dashboard shows them.
var Statuses = []string{StatusProvisioned, StatusFailed, StatusInitialized, StatusDecommissioned}

// LinkKind says why two nodes are linked.
type LinkKind int

const (
	// LinkDependency points from the producer of a resource to its consumer.
	LinkDependency LinkKind = iota
	// LinkFork points from an environment to its fork.
	LinkFork
)

// Graph is the constellation: environments linked by the resources they share
// and the forks they came from.
type Graph struct {
	Envs      []*EnvNode
	EnvLinks  []Link
	InstLinks []Link
	Projects  []string
	Firing    []FiringAlarm

	envs      map[string]*EnvNode
	instances map[string]*InstNode
}

// EnvNode is one environment in the graph. Restricted environments hold
// resources that visible instances use but that you cannot view.
type EnvNode struct {
	ID          string
	Name        string
	ProjectID   string
	ProjectName string
	ParentID    string
	Restricted  bool
	Instances   []*InstNode
	Firing      []FiringAlarm
	Cost        float64
	Currency    string
	Banner      *Banner
	Links       []HeaderLink
}

// InstNode is one instance in the graph.
type InstNode struct {
	ID             string
	EnvID          string
	Status         string
	Bundle         string
	Version        string
	Runtime        string
	RuntimeVersion string
	Firing         int
	Upgrade        string
	Deployed       string
	Deploy         *Deployment
	Health         string
	Usage          *Usage
	Traffic        *Traffic
}

// IsApp reports whether the instance runs an application.
func (i *InstNode) IsApp() bool { return i.Runtime != "" }

// HasUpgrade reports whether a newer bundle version fits the instance's
// version constraint.
func (i *InstNode) HasUpgrade() bool {
	return i.Upgrade != "" && i.Deployed != "" && CompareVersions(i.Upgrade, i.Deployed) > 0
}

// Deploying reports whether a provision or decommission of the instance is
// running or queued. A plan changes nothing, so it does not count.
func (i *InstNode) Deploying() bool {
	return i.Deploy != nil && i.Deploy.Action != "PLAN" && (i.Deploy.Status == "RUNNING" || i.Deploy.Status == "PENDING")
}

// AwaitingApproval reports whether the latest deployment waits for someone
// to approve it.
func (i *InstNode) AwaitingApproval() bool {
	return i.Deploy != nil && i.Deploy.Status == "PROPOSED"
}

// Deploying lists the environment's instances with a deployment in flight.
func (e *EnvNode) Deploying() []*InstNode {
	var out []*InstNode
	for _, inst := range e.Instances {
		if inst.Deploying() {
			out = append(out, inst)
		}
	}
	return out
}

// FiringCount counts every firing alarm in the graph, including ones beyond
// the first page of an instance's alarms.
func (g *Graph) FiringCount() int {
	n := 0
	for _, env := range g.Envs {
		n += env.FiringCount()
	}
	return n
}

// FiringCount counts the environment's firing alarms.
func (e *EnvNode) FiringCount() int {
	n := 0
	for _, inst := range e.Instances {
		n += inst.Firing
	}
	return n
}

// Upgrades counts the environment's instances that can upgrade.
func (e *EnvNode) Upgrades() int {
	n := 0
	for _, inst := range e.Instances {
		if inst.HasUpgrade() {
			n++
		}
	}
	return n
}

// Link joins two nodes. For dependencies From produces and To consumes.
type Link struct {
	From  string
	To    string
	Kind  LinkKind
	Count int
}

// FiringAlarm is an alarm whose latest state is ALARM.
type FiringAlarm struct {
	InstanceID string
	EnvID      string
	Name       string
	Message    string
	Since      time.Time
}

// Env returns the environment with id, or nil.
func (g *Graph) Env(id string) *EnvNode { return g.envs[id] }

// Instance returns the instance with id, or nil.
func (g *Graph) Instance(id string) *InstNode { return g.instances[id] }

// StatusCount counts the environment's instances with status.
func (e *EnvNode) StatusCount(status string) int {
	n := 0
	for _, inst := range e.Instances {
		if inst.Status == status {
			n++
		}
	}
	return n
}

// Health is the status that best describes the environment as a whole.
func (e *EnvNode) Health() string {
	switch {
	case e.FiringCount() > 0 || e.StatusCount(StatusFailed) > 0:
		return StatusFailed
	case e.StatusCount(StatusProvisioned) > 0:
		return StatusProvisioned
	case e.StatusCount(StatusInitialized) > 0:
		return StatusInitialized
	case len(e.Instances) > 0:
		return StatusDecommissioned
	default:
		return StatusInitialized
	}
}

// BuildGraph turns a snapshot into the constellation.
func BuildGraph(snap *Snapshot) *Graph {
	g := &Graph{envs: map[string]*EnvNode{}, instances: map[string]*InstNode{}}
	for _, env := range snap.Environments {
		g.addEnv(env)
	}
	g.linkDependencies(snap.Environments)
	g.linkForks()
	g.collectProjects()
	g.applyWidgets(snap.Widgets)
	slices.SortFunc(g.Firing, func(a, b FiringAlarm) int { return b.Since.Compare(a.Since) })
	return g
}

func (g *Graph) addEnv(env Environment) {
	node := &EnvNode{ID: env.ID, Name: env.Name}
	if env.Project != nil {
		node.ProjectID, node.ProjectName = env.Project.ID, env.Project.Name
	}
	if env.Parent != nil {
		node.ParentID = env.Parent.ID
	}
	for _, inst := range env.Instances {
		in := toInstNode(env.ID, inst)
		node.Instances = append(node.Instances, in)
		g.instances[in.ID] = in
		firing := firingAlarms(env.ID, inst)
		// The alarm list is one page; the count covers every alarm.
		in.Firing = max(len(firing), inst.FiringAlarmCount)
		node.Firing = append(node.Firing, firing...)
		if amount := inst.Cost.LastMonth.Amount; amount != nil {
			node.Cost += *amount
			node.Currency = inst.Cost.LastMonth.Currency
		}
	}
	g.Firing = append(g.Firing, node.Firing...)
	g.Envs = append(g.Envs, node)
	g.envs[node.ID] = node
}

func toInstNode(envID string, inst Instance) *InstNode {
	in := &InstNode{
		ID:       inst.ID,
		EnvID:    envID,
		Status:   inst.Status,
		Deployed: inst.DeployedVersion,
		Upgrade:  inst.AvailableUpgrade,
		Deploy:   inst.LatestDeployment,
	}
	b := inst.Bundle
	if inst.LatestProvision != nil && inst.LatestProvision.Bundle != nil {
		b = inst.LatestProvision.Bundle
	}
	if b != nil {
		in.Bundle, in.Version = b.Name, b.Version
		if b.Runtime != nil {
			in.Runtime, in.RuntimeVersion = b.Runtime.Name, b.Runtime.Version
		}
	}
	return in
}

func firingAlarms(envID string, inst Instance) []FiringAlarm {
	var out []FiringAlarm
	for _, alarm := range inst.Alarms.Items {
		state := alarm.CurrentState
		if state == nil || state.Status != "ALARM" {
			continue
		}
		since, _ := time.Parse(time.RFC3339, state.OccurredAt)
		out = append(out, FiringAlarm{
			InstanceID: inst.ID,
			EnvID:      envID,
			Name:       alarm.DisplayName,
			Message:    state.Message,
			Since:      since,
		})
	}
	return out
}

func (g *Graph) linkDependencies(envs []Environment) {
	envLinks := map[[2]string]int{}
	instLinks := map[[2]string]int{}
	for _, env := range envs {
		for _, inst := range env.Instances {
			for _, dep := range inst.Dependencies {
				producer := dep.Resource.Instance
				if producer == nil || producer.Environment == nil || producer.ID == inst.ID {
					continue
				}
				g.ensureEnv(producer.Environment.ID)
				instLinks[[2]string{producer.ID, inst.ID}]++
				if producer.Environment.ID != env.ID {
					envLinks[[2]string{producer.Environment.ID, env.ID}]++
				}
			}
		}
	}
	g.EnvLinks = sortedLinks(envLinks, LinkDependency)
	g.InstLinks = sortedLinks(instLinks, LinkDependency)
}

func (g *Graph) ensureEnv(id string) {
	if _, ok := g.envs[id]; ok {
		return
	}
	node := &EnvNode{ID: id, Name: id, Restricted: true}
	g.Envs = append(g.Envs, node)
	g.envs[id] = node
}

func (g *Graph) linkForks() {
	for _, env := range g.Envs {
		if env.ParentID != "" && g.envs[env.ParentID] != nil {
			g.EnvLinks = append(g.EnvLinks, Link{From: env.ParentID, To: env.ID, Kind: LinkFork, Count: 1})
		}
	}
}

func (g *Graph) collectProjects() {
	seen := map[string]bool{}
	for _, env := range g.Envs {
		if env.ProjectID != "" && !seen[env.ProjectID] {
			seen[env.ProjectID] = true
			g.Projects = append(g.Projects, env.ProjectID)
		}
	}
	slices.Sort(g.Projects)
}

func sortedLinks(counts map[[2]string]int, kind LinkKind) []Link {
	links := make([]Link, 0, len(counts))
	for key, n := range counts {
		links = append(links, Link{From: key[0], To: key[1], Kind: kind, Count: n})
	}
	slices.SortFunc(links, func(a, b Link) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	return links
}

// Producers lists the environments that envID uses resources from.
func (g *Graph) Producers(envID string) []Link {
	return g.filterLinks(func(l Link) bool { return l.Kind == LinkDependency && l.To == envID })
}

// Consumers lists the environments that use resources from envID.
func (g *Graph) Consumers(envID string) []Link {
	return g.filterLinks(func(l Link) bool { return l.Kind == LinkDependency && l.From == envID })
}

func (g *Graph) filterLinks(keep func(Link) bool) []Link {
	var out []Link
	for _, l := range g.EnvLinks {
		if keep(l) {
			out = append(out, l)
		}
	}
	return out
}

// Blast is how far trouble in one place spreads: every node downstream of the
// source, keyed by the number of dependency hops it sits from the source.
type Blast struct {
	Envs      map[string]int
	Instances map[string]int
}

// Reaches reports whether the blast reaches id, and in how many hops.
func (b *Blast) Reaches(id string) (int, bool) {
	if b == nil {
		return 0, false
	}
	if hop, ok := b.Envs[id]; ok {
		return hop, true
	}
	hop, ok := b.Instances[id]
	return hop, ok
}

// Downstream counts the environments and instances the blast reaches beyond
// the source.
func (b *Blast) Downstream() (int, int) {
	envs, insts := 0, 0
	for _, hop := range b.Envs {
		if hop > 0 {
			envs++
		}
	}
	for _, hop := range b.Instances {
		if hop > 0 {
			insts++
		}
	}
	return envs, insts
}

// BlastRadius follows dependency links from the given instances, or from every
// instance of the given environments, to everything that consumes them.
func (g *Graph) BlastRadius(sourceIDs ...string) *Blast {
	consumers := map[string][]string{}
	for _, l := range g.InstLinks {
		consumers[l.From] = append(consumers[l.From], l.To)
	}
	hops := map[string]int{}
	var frontier []string
	for _, id := range sourceIDs {
		for _, seed := range g.seeds(id) {
			if _, ok := hops[seed]; !ok {
				hops[seed] = 0
				frontier = append(frontier, seed)
			}
		}
	}
	for hop := 1; len(frontier) > 0; hop++ {
		var next []string
		for _, id := range frontier {
			for _, c := range consumers[id] {
				if _, ok := hops[c]; !ok {
					hops[c] = hop
					next = append(next, c)
				}
			}
		}
		frontier = next
	}
	return &Blast{Envs: g.envHops(hops, sourceIDs), Instances: hops}
}

func (g *Graph) seeds(id string) []string {
	if env := g.envs[id]; env != nil {
		ids := make([]string, 0, len(env.Instances))
		for _, inst := range env.Instances {
			ids = append(ids, inst.ID)
		}
		return ids
	}
	if g.instances[id] != nil {
		return []string{id}
	}
	return nil
}

func (g *Graph) envHops(instHops map[string]int, sourceIDs []string) map[string]int {
	out := map[string]int{}
	for _, id := range sourceIDs {
		if g.envs[id] != nil {
			out[id] = 0
		}
	}
	for id, hop := range instHops {
		inst := g.instances[id]
		if inst == nil {
			continue
		}
		if cur, ok := out[inst.EnvID]; !ok || hop < cur {
			out[inst.EnvID] = hop
		}
	}
	return out
}

// Layers ranks environments so every producer sits above its consumers:
// environments that use nothing are layer 0, and each consumer sits one layer
// below its deepest producer.
func (g *Graph) Layers() map[string]int {
	layers := map[string]int{}
	for _, env := range g.Envs {
		layers[env.ID] = 0
	}
	for range g.Envs {
		changed := false
		for _, l := range g.EnvLinks {
			if l.Kind == LinkDependency && layers[l.To] < layers[l.From]+1 {
				layers[l.To] = layers[l.From] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return layers
}
