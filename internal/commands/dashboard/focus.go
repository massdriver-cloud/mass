package dashboard

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// DefaultDepth is how many hops from the focus the dashboard loads.
const DefaultDepth = 1

// maxEnvironmentIDs is the most environments one read asks for by id.
const maxEnvironmentIDs = 50

// Place is an environment as the directory lists it: enough to find it.
type Place struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Project *Ref   `json:"project"`
}

// InstanceResource is a resource an instance produces.
type InstanceResource struct {
	Resource ProducedResource `json:"resource"`
}

// ProducedResource is who uses a resource from outside its environment:
// instances that reference it, and environments that default to it.
type ProducedResource struct {
	RemoteReferences    RemoteReferences    `json:"remoteReferences"`
	EnvironmentDefaults EnvironmentDefaults `json:"environmentDefaults"`
}

// RemoteReferences are the instances that reference a resource.
type RemoteReferences struct {
	Items []RemoteReference `json:"items"`
}

// RemoteReference is one instance that references a resource.
type RemoteReference struct {
	Instance *ResourceInstance `json:"instance"`
}

// EnvironmentDefaults are the environments that default to a resource.
type EnvironmentDefaults struct {
	Items []EnvironmentDefault `json:"items"`
}

// EnvironmentDefault is one environment that defaults to a resource.
type EnvironmentDefault struct {
	Environment *Ref `json:"environment"`
}

const environmentFields = `
      id name createdAt
      project { id name }
      parent { id }
      instances {
        id status deployedVersion availableUpgrade firingAlarmCount
        component { id }
        bundle { name version runtime { name version } }
        latestProvision { createdAt bundle { name version runtime { name version } } }
        latestDeployment { status action createdAt }
        cost { lastMonth { amount currency } }
        alarms { items { displayName currentState { status message occurredAt } } }
        dependencies { field resource { id origin instance { id environment { id } } } }
        resources {
          resource {
            remoteReferences(cursor: {limit: 100}) { items { instance { id environment { id } } } }
            environmentDefaults(cursor: {limit: 100}) { items { environment { id } } }
          }
        }
      }`

const environmentsByIDQuery = `query dashboardEnvironments($organizationId: ID!, $ids: [ID!]!, $cursor: Cursor) {
  environments(organizationId: $organizationId, filter: {id: {in: $ids}}, cursor: $cursor) {
    cursor { next }
    items {` + environmentFields + `
    }
  }
}`

const directoryQuery = `query dashboardDirectory($organizationId: ID!, $cursor: Cursor) {
  environments(organizationId: $organizationId, sort: {field: NAME, order: ASC}, cursor: $cursor) {
    cursor { next }
    items { id name project { id name } }
  }
}`

const recentQuery = `query dashboardRecent($organizationId: ID!) {
  deployments(organizationId: $organizationId, cursor: {limit: 1}) {
    items { instance { environment { id } } }
  }
  environments(organizationId: $organizationId, sort: {field: CREATED_AT, order: DESC}, cursor: {limit: 1}) {
    items { id }
  }
}`

func (a *gqlAPI) EnvironmentsByID(ctx context.Context, ids []string, cursor string) ([]Environment, string, error) {
	cur := map[string]any{"limit": pageLimit}
	if cursor != "" {
		cur["next"] = cursor
	}
	vars := map[string]any{"organizationId": a.orgID, "ids": ids, "cursor": cur}
	var data struct {
		Page page[Environment] `json:"environments"`
	}
	err := a.request(ctx, "dashboardEnvironments", environmentsByIDQuery, vars, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

func (a *gqlAPI) Directory(ctx context.Context, cursor string) ([]Place, string, error) {
	var data struct {
		Page page[Place] `json:"environments"`
	}
	err := a.query(ctx, "dashboardDirectory", directoryQuery, cursor, &data)
	return data.Page.Items, data.Page.Cursor.Next, err
}

// RecentEnvironment is the environment of the organization's most recent
// deployment, or its newest environment when nothing has deployed.
func (a *gqlAPI) RecentEnvironment(ctx context.Context) (string, error) {
	var data struct {
		Deployments struct {
			Items []struct {
				Instance *ResourceInstance `json:"instance"`
			} `json:"items"`
		} `json:"deployments"`
		Environments struct {
			Items []Ref `json:"items"`
		} `json:"environments"`
	}
	if err := a.request(ctx, "dashboardRecent", recentQuery, map[string]any{"organizationId": a.orgID}, &data); err != nil {
		return "", err
	}
	for _, d := range data.Deployments.Items {
		if d.Instance != nil && d.Instance.Environment != nil {
			return d.Instance.Environment.ID, nil
		}
	}
	for _, e := range data.Environments.Items {
		return e.ID, nil
	}
	return "", nil
}

// ErrNoEnvironments means the organization has no environment to show.
var ErrNoEnvironments = errors.New("no environments to show")

// loadAround reads the focus environment and every environment within depth
// hops of it, ring by ring: the producers whose resources it uses, the
// consumers that use its resources, and the environment it was forked from.
// Missing lists the ids that were asked for and not returned, which the
// caller cannot view.
func loadAround(ctx context.Context, api API, focus string, depth int) ([]Environment, []string, error) {
	if depth <= 0 {
		depth = DefaultDepth
	}
	r := reach{loaded: map[string]Environment{}}
	ring := []string{focus}
	for hop := 0; hop <= depth && len(ring) > 0; hop++ {
		envs, err := loadIDs(ctx, api, ring)
		if err != nil {
			return nil, nil, err
		}
		if hop == 0 && len(envs) == 0 {
			return nil, nil, fmt.Errorf("environment %q: not found, or you cannot view it", focus)
		}
		ring = r.add(ring, envs)
	}
	out := make([]Environment, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.loaded[id])
	}
	return out, r.missing, nil
}

type reach struct {
	loaded  map[string]Environment
	order   []string
	missing []string
}

// add takes one ring's reads, notes the ids that did not come back, and
// returns the next ring out.
func (r *reach) add(asked []string, envs []Environment) []string {
	found := map[string]bool{}
	for _, env := range envs {
		found[env.ID] = true
		r.loaded[env.ID] = env
		r.order = append(r.order, env.ID)
	}
	for _, id := range asked {
		if !found[id] {
			r.missing = append(r.missing, id)
		}
	}
	var next []string
	for _, env := range envs {
		for _, id := range neighbors(env) {
			if _, ok := r.loaded[id]; !ok && !slices.Contains(next, id) && !slices.Contains(r.missing, id) {
				next = append(next, id)
			}
		}
	}
	return next
}

func loadIDs(ctx context.Context, api API, ids []string) ([]Environment, error) {
	var out []Environment
	for start := 0; start < len(ids); start += maxEnvironmentIDs {
		batch := ids[start:min(start+maxEnvironmentIDs, len(ids))]
		page := func(ctx context.Context, cursor string) ([]Environment, string, error) {
			return api.EnvironmentsByID(ctx, batch, cursor)
		}
		envs, err := drainAs(ctx, "read environments", page)
		if err != nil {
			return nil, err
		}
		out = append(out, envs...)
	}
	return out, nil
}

func neighbors(env Environment) []string {
	var out []string
	add := func(id string) {
		if id != "" && id != env.ID && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	if env.Parent != nil {
		add(env.Parent.ID)
	}
	for _, inst := range env.Instances {
		for _, dep := range inst.Dependencies {
			if p := dep.Resource.Instance; p != nil && p.Environment != nil {
				add(p.Environment.ID)
			}
		}
		for _, consumer := range consumers(inst) {
			add(consumer.env)
		}
	}
	return out
}

type consumer struct{ instance, env string }

func consumers(inst Instance) []consumer {
	var out []consumer
	for _, r := range inst.Resources {
		for _, ref := range r.Resource.RemoteReferences.Items {
			if ref.Instance != nil && ref.Instance.Environment != nil {
				out = append(out, consumer{instance: ref.Instance.ID, env: ref.Instance.Environment.ID})
			}
		}
		for _, def := range r.Resource.EnvironmentDefaults.Items {
			if def.Environment != nil {
				out = append(out, consumer{env: def.Environment.ID})
			}
		}
	}
	return out
}

// FindPlace picks the environment a search means: an exact id first, then a
// name or project that starts with the query, then one that contains it.
func FindPlace(places []Place, query string) (Place, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return Place{}, false
	}
	label := func(p Place) string {
		project := ""
		if p.Project != nil {
			project = p.Project.Name + " " + p.Project.ID
		}
		return strings.ToLower(p.ID + " " + p.Name + " " + project)
	}
	for _, rank := range []func(Place) bool{
		func(p Place) bool { return strings.ToLower(p.ID) == q },
		func(p Place) bool { return strings.HasPrefix(strings.ToLower(p.ID), q) },
		func(p Place) bool { return strings.HasPrefix(strings.ToLower(p.Name), q) },
		func(p Place) bool { return strings.Contains(label(p), q) },
	} {
		for _, p := range places {
			if rank(p) {
				return p, true
			}
		}
	}
	return Place{}, false
}
