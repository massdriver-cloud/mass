// Package dashboard holds the logic behind `mass dashboard`: a live terminal
// view of an organization's environments, the resources they share, their
// firing alarms, and the runtimes their applications build on.
package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/auditlogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/streaming"
	"golang.org/x/sync/errgroup"
)

// API is the narrow surface the dashboard reads from. Each call returns one
// page and the cursor for the next, empty on the last page.
type API interface {
	EnvironmentsByID(ctx context.Context, ids []string, cursor string) ([]Environment, string, error)
	Directory(ctx context.Context, cursor string) ([]Place, string, error)
	RecentEnvironment(ctx context.Context) (string, error)
	Runtimes(ctx context.Context, cursor string) ([]Runtime, string, error)
	Applications(ctx context.Context, cursor string) ([]Application, string, error)
	StashEntries(ctx context.Context, scopes []Scope, cursor string) ([]StashEntry, string, error)
	// SubscribeStash streams the keys set or deleted in an environment's
	// stashes: its project, the environment, its components, and its
	// instances.
	SubscribeStash(ctx context.Context, projectID, environmentID string) (*streaming.Stream[StashEvent], error)
	Annotations(ctx context.Context, projectID, cursor string) ([]Annotation, string, error)
	Events(ctx context.Context, since time.Time) (Log, error)
	History(ctx context.Context, since time.Time) (Log, error)
	DeploymentLogs(ctx context.Context, deploymentID string) (DeploymentLog, error)
	StreamDeploymentLogs(ctx context.Context, deploymentID string) (*streaming.Stream[LogBatch], error)
	// Deployments reads the deployments of instances in the given statuses,
	// newest first.
	Deployments(ctx context.Context, q DeploymentQuery, cursor string) ([]Deployment, string, error)
	// PeekDeployment reads a deployment's configuration and what it changes
	// since the instance's last completed provision.
	PeekDeployment(ctx context.Context, instanceID, deploymentID string) (DeploymentPeek, error)
	ApproveDeployment(ctx context.Context, deploymentID string) error
	// RejectDeployment rejects a proposed deployment, and keeps a non-empty
	// reason in its stash.
	RejectDeployment(ctx context.Context, deploymentID, reason string) error
	// Deploy provisions an instance again with its current configuration.
	Deploy(ctx context.Context, instanceID string) error
}

// AuditLister reads one page of the audit log.
type AuditLister interface {
	ListPage(ctx context.Context, input auditlogs.ListInput) (types.Page[types.AuditLog], error)
}

// Event is one entry in the audit log.
type Event struct {
	ID     string    `json:"id"`
	At     time.Time `json:"at"`
	Type   string    `json:"type"`
	EnvID  string    `json:"envId,omitempty"`
	Actor  string    `json:"actor,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

// Environment is one environment with the instances it holds.
type Environment struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt string     `json:"createdAt"`
	Project   *Ref       `json:"project"`
	Parent    *Ref       `json:"parent"`
	Instances []Instance `json:"instances"`
}

// Ref names an object by id and, where the API gives one, name.
type Ref struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Instance is one instance and what it depends on.
type Instance struct {
	ID               string             `json:"id"`
	Status           string             `json:"status"`
	DeployedVersion  string             `json:"deployedVersion"`
	AvailableUpgrade string             `json:"availableUpgrade"`
	FiringAlarmCount int                `json:"firingAlarmCount"`
	Component        *Ref               `json:"component"`
	Resources        []InstanceResource `json:"resources"`
	Bundle           *Bundle            `json:"bundle"`
	LatestProvision  *Provision         `json:"latestProvision"`
	LatestDeployment *Deployment        `json:"latestDeployment"`
	Alarms           AlarmsPage         `json:"alarms"`
	Dependencies     []Dependency       `json:"dependencies"`
	Cost             Cost               `json:"cost"`
}

// Bundle is the bundle version an instance or application uses. Runtime is
// set when the bundle is an application.
type Bundle struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Runtime *Bundle `json:"runtime"`
}

// Provision is an instance's latest completed provision.
type Provision struct {
	CreatedAt string  `json:"createdAt"`
	Bundle    *Bundle `json:"bundle"`
}

// Deployment is one deployment of an instance. The map reads only each
// instance's latest, without its message, author, or timing.
type Deployment struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Action      string `json:"action"`
	CreatedAt   string `json:"createdAt"`
	Version     string `json:"version,omitempty"`
	Message     string `json:"message,omitempty"`
	DeployedBy  string `json:"deployedBy,omitempty"`
	ElapsedTime int    `json:"elapsedTime,omitempty"`
	Instance    *Ref   `json:"instance,omitempty"`
}

// AlarmsPage is the first page of an instance's alarms.
type AlarmsPage struct {
	Items []Alarm `json:"items"`
}

// Alarm is one cloud metric alarm and its latest reported state.
type Alarm struct {
	DisplayName  string      `json:"displayName"`
	CurrentState *AlarmState `json:"currentState"`
}

// AlarmState is the latest state an alarm reported.
type AlarmState struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurredAt"`
}

// Dependency is one filled dependency slot and the resource that fills it.
type Dependency struct {
	Field    string   `json:"field"`
	Resource Resource `json:"resource"`
}

// Resource is the resource behind a dependency. Instance is nil for imported
// resources and for resources whose producing project you cannot view.
type Resource struct {
	ID       string            `json:"id"`
	Origin   string            `json:"origin"`
	Instance *ResourceInstance `json:"instance"`
}

// ResourceInstance is the instance that produced a resource.
type ResourceInstance struct {
	ID          string `json:"id"`
	Environment *Ref   `json:"environment"`
}

// Cost is an instance's spend for the last month.
type Cost struct {
	LastMonth CostSample `json:"lastMonth"`
}

// CostSample is one cost figure. Amount is nil when no cost was reported.
type CostSample struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency"`
}

// Runtime is a bundle that applications build on.
type Runtime struct {
	ID       string     `json:"id"`
	Latest   Bundle     `json:"latest"`
	Adoption []Adoption `json:"adoption"`
}

// Adoption counts the applications and instances on one runtime version.
type Adoption struct {
	Version          string `json:"version"`
	ApplicationCount int    `json:"applicationCount"`
	InstanceCount    int    `json:"instanceCount"`
}

// Application is a bundle that runs on a runtime.
type Application struct {
	ID     string `json:"id"`
	Latest Bundle `json:"latest"`
}

// Snapshot is everything the dashboard shows, read at one moment.
type Snapshot struct {
	// Focus is the environment the view is centered on, and Environments the
	// ones within reach of it.
	Focus        string
	Environments []Environment
	// Missing are environments within reach that could not be read, because
	// they are gone or you cannot view them.
	Missing      []string
	Runtimes     []Runtime
	Applications []Application
	Annotations  []Annotation
	// AnnotationsErr is why annotations could not be read, such as a
	// service account, which has none.
	AnnotationsErr error
}

// Where says what to center the dashboard on and how far out to read.
type Where struct {
	// Focus is an environment id; empty means the most recently deployed.
	Focus string
	Depth int
}

// Fetch reads the environments within reach of the focus, and every
// runtime and application.
func Fetch(ctx context.Context, api API, where Where) (*Snapshot, error) {
	snap := &Snapshot{Focus: where.Focus}
	if snap.Focus == "" {
		recent, err := api.RecentEnvironment(ctx)
		if err != nil {
			return nil, fmt.Errorf("find the most recent environment: %w", err)
		}
		if recent == "" {
			return nil, ErrNoEnvironments
		}
		snap.Focus = recent
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		snap.Environments, snap.Missing, err = loadAround(groupCtx, api, snap.Focus, where.Depth)
		return err
	})
	group.Go(func() (err error) {
		snap.Runtimes, err = drainAs(groupCtx, "list runtimes", api.Runtimes)
		return err
	})
	group.Go(func() (err error) {
		snap.Applications, err = drainAs(groupCtx, "list applications", api.Applications)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}
	snap.Annotations, snap.AnnotationsErr = readAnnotations(ctx, api, snap.Environments)
	cleanSnapshot(snap)
	return snap, nil
}

// cleanSnapshot strips control characters from the bundle and version text
// that publishers write, so it reaches the terminal as text.
func cleanSnapshot(snap *Snapshot) {
	for i := range snap.Runtimes {
		rt := &snap.Runtimes[i]
		rt.ID = clean(rt.ID)
		cleanBundle(&rt.Latest)
		for j := range rt.Adoption {
			rt.Adoption[j].Version = clean(rt.Adoption[j].Version)
		}
	}
	for i := range snap.Applications {
		app := &snap.Applications[i]
		app.ID = clean(app.ID)
		cleanBundle(&app.Latest)
	}
	for i := range snap.Environments {
		for j := range snap.Environments[i].Instances {
			inst := &snap.Environments[i].Instances[j]
			inst.DeployedVersion, inst.AvailableUpgrade = clean(inst.DeployedVersion), clean(inst.AvailableUpgrade)
			cleanBundle(inst.Bundle)
			if inst.LatestProvision != nil {
				cleanBundle(inst.LatestProvision.Bundle)
			}
		}
	}
}

func cleanBundle(b *Bundle) {
	if b == nil {
		return
	}
	b.Name, b.Version = clean(b.Name), clean(b.Version)
	cleanBundle(b.Runtime)
}

func drainAs[T any](ctx context.Context, what string, page func(context.Context, string) ([]T, string, error)) ([]T, error) {
	items, err := drain(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", what, err)
	}
	return items, nil
}

// maxPages stops a server that keeps handing back a cursor from looping forever.
const maxPages = 1000

func drain[T any](ctx context.Context, page func(context.Context, string) ([]T, string, error)) ([]T, error) {
	var all []T
	cursor := ""
	for range maxPages {
		items, next, err := page(ctx, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if next == "" {
			return all, nil
		}
		cursor = next
	}
	return all, nil
}

type gqlAPI struct {
	client   graphql.Client
	audit    AuditLister
	stash    StashService
	deploys  DeploymentService
	deployer Deployer
	orgID    string

	mu sync.Mutex
	// limits is the page size each operation settled on after the API
	// refused a page as too complex, so later pages start there.
	limits map[string]int
}

// NewAPI returns the production [API], sending queries through the SDK's
// GraphQL transport for the client's profile.
func NewAPI(c *massdriver.Client) API {
	cfg := c.Config()
	return &gqlAPI{client: gql.NewV2Client(cfg), audit: c.AuditLogs, stash: c.Stash, deploys: c.Deployments, deployer: NewDeployer(c), orgID: cfg.OrganizationID}
}

// Services are the SDK services the [API] reads and acts through, besides
// GraphQL.
type Services struct {
	Audit    AuditLister
	Stash    StashService
	Deploys  DeploymentService
	Deployer Deployer
}

// NewAPIWithClient returns an [API] that sends queries through client and
// reaches everything else through services.
func NewAPIWithClient(client graphql.Client, services Services, orgID string) API {
	return &gqlAPI{client: client, audit: services.Audit, stash: services.Stash, deploys: services.Deploys, deployer: services.Deployer, orgID: orgID}
}

const (
	auditPageSize = 100
	// eventLimit bounds one poll; a burst bigger than this between polls
	// shows only its newest events.
	eventLimit = 500
	// historyLimit bounds a replay; a busier window keeps its newest events.
	historyLimit = 1000
)

// Log is a stretch of the audit log, oldest first. Truncated says the
// stretch held more events than were read, and the oldest were left out.
type Log struct {
	Events    []Event
	Truncated bool
}

func (a *gqlAPI) Events(ctx context.Context, since time.Time) (Log, error) {
	return a.readAudit(ctx, since, eventLimit)
}

func (a *gqlAPI) History(ctx context.Context, since time.Time) (Log, error) {
	return a.readAudit(ctx, since, historyLimit)
}

// readAudit pages back from the newest event, the audit log's default order.
func (a *gqlAPI) readAudit(ctx context.Context, since time.Time, limit int) (Log, error) {
	var log Log
	after := ""
	for {
		page, err := a.audit.ListPage(ctx, auditlogs.ListInput{TimeRangeStart: since, PageSize: auditPageSize, After: after})
		if err != nil {
			return Log{}, err
		}
		for _, entry := range page.Items {
			log.Events = append(log.Events, ToEvent(entry))
		}
		if len(log.Events) >= limit {
			log.Truncated = len(log.Events) > limit || page.Next != ""
			log.Events = log.Events[:limit]
			break
		}
		if page.Next == "" || len(page.Items) == 0 {
			break
		}
		after = page.Next
	}
	slices.Reverse(log.Events)
	return log, nil
}

// ToEvent reads the environment, actor, and a short detail out of an audit
// log entry. The subject is an MRI, mri:<org>:<project>:<env>:<kind>:<id>, and
// an environment's id is <project>-<env>.
func ToEvent(log types.AuditLog) Event {
	ev := Event{ID: log.ID, At: log.OccurredAt, Type: clean(log.Type)}
	if parts := strings.Split(log.Subject, ":"); len(parts) >= 4 && parts[2] != "" && parts[3] != "" {
		ev.EnvID = parts[2] + "-" + parts[3]
	}
	if log.Actor != nil && log.Actor.Type != string(auditlogs.ActorSystem) {
		ev.Actor = clean(log.Actor.Name)
	}
	entity, _ := log.Data["entity"].(map[string]any)
	switch {
	case entity["action"] != nil:
		ev.Detail = strings.ToLower(clean(fmt.Sprint(entity["action"])))
	case entity["name"] != nil:
		ev.Detail = clean(fmt.Sprint(entity["name"]))
	case entity["field"] != nil:
		ev.Detail = clean(fmt.Sprint(entity["field"]))
	}
	if ev.Actor == "" && entity["deployed_by_name"] != nil {
		ev.Actor = clean(fmt.Sprint(entity["deployed_by_name"]))
	}
	return ev
}

const runtimesQuery = `query dashboardRuntimes($organizationId: ID!, $cursor: Cursor) {
  runtimes(organizationId: $organizationId, cursor: $cursor) {
    cursor { next }
    items { id latest { name version } adoption { version applicationCount instanceCount } }
  }
}`

const applicationsQuery = `query dashboardApplications($organizationId: ID!, $cursor: Cursor) {
  applications(organizationId: $organizationId, cursor: $cursor) {
    cursor { next }
    items { id latest { name version runtime { name version } } }
  }
}`

const pageLimit = 50

type page[T any] struct {
	Cursor struct {
		Next string `json:"next"`
	} `json:"cursor"`
	Items []T `json:"items"`
}

func (a *gqlAPI) Runtimes(ctx context.Context, cursor string) ([]Runtime, string, error) {
	var data struct {
		Page page[Runtime] `json:"runtimes"`
	}
	err := a.query(ctx, "dashboardRuntimes", runtimesQuery, cursor, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

func (a *gqlAPI) Applications(ctx context.Context, cursor string) ([]Application, string, error) {
	var data struct {
		Page page[Application] `json:"applications"`
	}
	err := a.query(ctx, "dashboardApplications", applicationsQuery, cursor, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

func (a *gqlAPI) query(ctx context.Context, op, query, cursor string, out any) error {
	cur := map[string]any{"limit": pageLimit}
	if cursor != "" {
		cur["next"] = cursor
	}
	return a.request(ctx, op, query, map[string]any{"organizationId": a.orgID, "cursor": cur}, out)
}

// ErrTooComplex means a request costs more than the API allows, even one
// item at a time.
var ErrTooComplex = errors.New("the request is more complex than the API allows")

// request sends one operation. The API scores each request's complexity and
// refuses one over its limit, which a large page of environments, with their
// instances, alarms, and dependencies, can pass. A paged request that is
// refused asks again for a smaller page, sized from the score the API
// reports, and the operation keeps that page size.
func (a *gqlAPI) request(ctx context.Context, op, query string, vars map[string]any, out any) error {
	cur, paged := vars["cursor"].(map[string]any)
	for {
		if paged {
			cur["limit"] = a.limitFor(op)
		}
		err := a.send(ctx, op, query, vars, out)
		cost, most, refused := tooComplex(err)
		if !refused {
			return err
		}
		limit, _ := cur["limit"].(int)
		if !paged || limit <= 1 {
			return fmt.Errorf("%s: %w (complexity %d, maximum %d)", op, ErrTooComplex, cost, most)
		}
		// The margin saves a second refusal when later pages cost a little more.
		a.setLimit(op, max(1, min(limit-1, limit*most*9/(max(cost, 1)*10))))
	}
}

func (a *gqlAPI) send(ctx context.Context, op, query string, vars map[string]any, out any) error {
	var raw json.RawMessage
	resp := &graphql.Response{Data: &raw}
	if err := a.client.MakeRequest(ctx, &graphql.Request{OpName: op, Query: query, Variables: vars}, resp); err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (a *gqlAPI) limitFor(op string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	if limit, ok := a.limits[op]; ok {
		return limit
	}
	return pageLimit
}

func (a *gqlAPI) setLimit(op string, limit int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.limits == nil {
		a.limits = map[string]int{}
	}
	// Concurrent reads of one operation can be refused together; the smallest
	// page wins so a late answer does not raise the limit back.
	if held, ok := a.limits[op]; ok && held < limit {
		return
	}
	a.limits[op] = limit
}

var complexityError = regexp.MustCompile(`complexity is (\d+) and maximum is (\d+)`)

func tooComplex(err error) (int, int, bool) {
	if err == nil {
		return 0, 0, false
	}
	m := complexityError.FindStringSubmatch(err.Error())
	if m == nil {
		return 0, 0, false
	}
	cost, _ := strconv.Atoi(m[1])
	most, _ := strconv.Atoi(m[2])
	return cost, most, true
}
