// Package dashboard holds the logic behind `mass dashboard`: a live terminal
// view of an organization's environments, the resources they share, their
// firing alarms, and the runtimes their applications build on.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/gql"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/auditlogs"
	"github.com/massdriver-cloud/massdriver-sdk-go/massdriver/platform/types"
)

// API is the narrow surface the dashboard reads from. Each call returns one
// page and the cursor for the next, empty on the last page.
type API interface {
	Environments(ctx context.Context, cursor string) ([]Environment, string, error)
	Runtimes(ctx context.Context, cursor string) ([]Runtime, string, error)
	Applications(ctx context.Context, cursor string) ([]Application, string, error)
	Events(ctx context.Context, since time.Time) ([]Event, error)
	History(ctx context.Context, since time.Time) ([]Event, error)
}

// AuditLister reads one page of the audit log.
type AuditLister interface {
	ListPage(ctx context.Context, input auditlogs.ListInput) (types.Page[types.AuditLog], error)
}

// Event is one entry in the audit log.
type Event struct {
	ID     string
	At     time.Time
	Type   string
	EnvID  string
	Actor  string
	Detail string
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
	ID               string       `json:"id"`
	Status           string       `json:"status"`
	DeployedVersion  string       `json:"deployedVersion"`
	AvailableUpgrade string       `json:"availableUpgrade"`
	Bundle           *Bundle      `json:"bundle"`
	LatestProvision  *Provision   `json:"latestProvision"`
	LatestDeployment *Deployment  `json:"latestDeployment"`
	Alarms           AlarmsPage   `json:"alarms"`
	Dependencies     []Dependency `json:"dependencies"`
	Cost             Cost         `json:"cost"`
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

// Deployment is an instance's most recent deployment of any kind.
type Deployment struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	CreatedAt string `json:"createdAt"`
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
	Environments []Environment
	Runtimes     []Runtime
	Applications []Application
}

// Fetch reads every page of environments, runtimes, and applications.
func Fetch(ctx context.Context, api API) (*Snapshot, error) {
	envs, err := drain(ctx, api.Environments)
	if err != nil {
		return nil, fmt.Errorf("list environments: %w", err)
	}
	runtimes, err := drain(ctx, api.Runtimes)
	if err != nil {
		return nil, fmt.Errorf("list runtimes: %w", err)
	}
	apps, err := drain(ctx, api.Applications)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	return &Snapshot{Environments: envs, Runtimes: runtimes, Applications: apps}, nil
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
	client graphql.Client
	audit  AuditLister
	orgID  string
}

// NewAPI returns the production [API], sending queries through the SDK's
// GraphQL transport for the client's profile.
func NewAPI(c *massdriver.Client) API {
	cfg := c.Config()
	return &gqlAPI{client: gql.NewV2Client(cfg), audit: c.AuditLogs, orgID: cfg.OrganizationID}
}

// NewAPIWithClient returns an [API] that sends queries through client and
// reads the audit log through audit.
func NewAPIWithClient(client graphql.Client, audit AuditLister, orgID string) API {
	return &gqlAPI{client: client, audit: audit, orgID: orgID}
}

const eventPageSize = 50

func (a *gqlAPI) Events(ctx context.Context, since time.Time) ([]Event, error) {
	page, err := a.audit.ListPage(ctx, auditlogs.ListInput{TimeRangeStart: since, PageSize: eventPageSize})
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(page.Items))
	for _, log := range page.Items {
		events = append(events, ToEvent(log))
	}
	return events, nil
}

// historyLimit caps how much of the audit log a replay reads.
const historyLimit = 1000

// History reads the audit log since a time, oldest first, up to historyLimit
// events; past the limit it keeps the newest.
func (a *gqlAPI) History(ctx context.Context, since time.Time) ([]Event, error) {
	var events []Event
	after := ""
	for len(events) < historyLimit {
		page, err := a.audit.ListPage(ctx, auditlogs.ListInput{TimeRangeStart: since, PageSize: 100, After: after})
		if err != nil {
			return nil, err
		}
		for _, log := range page.Items {
			events = append(events, ToEvent(log))
		}
		if page.Next == "" || len(page.Items) == 0 {
			break
		}
		after = page.Next
	}
	events = events[:min(len(events), historyLimit)]
	slices.Reverse(events)
	return events, nil
}

// ToEvent reads the environment, actor, and a short detail out of an audit
// log entry. The subject is an MRI, mri:<org>:<project>:<env>:<kind>:<id>, and
// an environment's id is <project>-<env>.
func ToEvent(log types.AuditLog) Event {
	ev := Event{ID: log.ID, At: log.OccurredAt, Type: log.Type}
	if parts := strings.Split(log.Subject, ":"); len(parts) >= 4 && parts[2] != "" && parts[3] != "" {
		ev.EnvID = parts[2] + "-" + parts[3]
	}
	if log.Actor != nil && log.Actor.Type != string(auditlogs.ActorSystem) {
		ev.Actor = log.Actor.Name
	}
	entity, _ := log.Data["entity"].(map[string]any)
	switch {
	case entity["action"] != nil:
		ev.Detail = strings.ToLower(fmt.Sprint(entity["action"]))
	case entity["name"] != nil:
		ev.Detail = fmt.Sprint(entity["name"])
	case entity["field"] != nil:
		ev.Detail = fmt.Sprint(entity["field"])
	}
	if ev.Actor == "" && entity["deployed_by_name"] != nil {
		ev.Actor = fmt.Sprint(entity["deployed_by_name"])
	}
	return ev
}

const environmentsQuery = `query dashboardEnvironments($organizationId: ID!, $cursor: Cursor) {
  environments(organizationId: $organizationId, sort: {field: CREATED_AT, order: DESC}, cursor: $cursor) {
    cursor { next }
    items {
      id name createdAt
      project { id name }
      parent { id }
      instances {
        id status deployedVersion availableUpgrade
        bundle { name version runtime { name version } }
        latestProvision { createdAt bundle { name version runtime { name version } } }
        latestDeployment { status action createdAt }
        cost { lastMonth { amount currency } }
        alarms { items { displayName currentState { status message occurredAt } } }
        dependencies { field resource { id origin instance { id environment { id } } } }
      }
    }
  }
}`

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

func (a *gqlAPI) Environments(ctx context.Context, cursor string) ([]Environment, string, error) {
	var data struct {
		Page page[Environment] `json:"environments"`
	}
	err := a.query(ctx, "dashboardEnvironments", environmentsQuery, cursor, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
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
	vars := map[string]any{"organizationId": a.orgID, "cursor": cur}
	var raw json.RawMessage
	resp := &graphql.Response{Data: &raw}
	if err := a.client.MakeRequest(ctx, &graphql.Request{OpName: op, Query: query, Variables: vars}, resp); err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
