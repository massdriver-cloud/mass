package dashboard_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func note(surface, typename, id, template string) dashboard.Annotation {
	return dashboard.Annotation{Kind: "NOTE", Surfaces: []string{surface}, Template: template,
		Entity: dashboard.AnnotationEntity{Typename: typename, ID: id}}
}

func withNotes() *fakeAPI {
	api := withStash()
	api.stash = append(api.stash,
		text(dashboard.Scope{Kind: "PROJECT", ID: "care"}, "release-train", "2026.12"),
		text(dashboard.Scope{Kind: "COMPONENT", ID: "care-app"}, "owner", "visits-team"),
	)
	api.envPages[0][1].Instances[1].Component = ref("care-queue", "")
	edge := note(dashboard.SurfaceCanvasEdge, "Link", "l1", "{{ source.stash[\"md/traffic\"].rps }} rps into {{ destination.id }}")
	edge.Entity.FromComponent, edge.Entity.ToComponent = ref("care-app", ""), ref("care-queue", "")
	api.annotations = map[string][]dashboard.Annotation{"care": {
		note(dashboard.SurfaceCanvasLegend, "Environment", "care-dev", "## Health: **{{ environment.id }}** ![chart](https://x/y.png)\n\nsee [runbook](https://wiki)"),
		note(dashboard.SurfaceCanvasNode, "Component", "care-app", "owned by {{ component.stash.owner }}, {{ instance.stash[\"md/health\"] }}"),
		note(dashboard.SurfaceProject, "Project", "care", "train {{ project.stash[\"release-train\"] }}"),
		note(dashboard.SurfaceCanvasLegend, "Environment", "care-dev", "{% if %}"),
		edge,
	}}
	return api
}

func TestNotes(t *testing.T) {
	g := graphOf(t, withNotes())
	care := g.Env("care-dev")
	if !equal(care.ProjectNotes, []string{"train 2026.12"}) {
		t.Errorf("project notes = %q", care.ProjectNotes)
	}
	if len(care.Notes) != 3 || care.Notes[0] != "Health: care-dev" || care.Notes[1] != "see runbook" ||
		!strings.HasPrefix(care.Notes[2], "(note does not render") {
		t.Errorf("environment notes = %q, want markdown reduced to text and a broken template named", care.Notes)
	}
	if got := g.Instance("care-dev-app").Notes; !equal(got, []string{"owned by visits-team, degraded"}) {
		t.Errorf("instance notes = %q, want a component note on its instance", got)
	}
	if len(care.LinkNotes) != 1 || care.LinkNotes[0].Text != "40 rps into care-dev-queue" || care.LinkNotes[0].From != "care-dev-app" {
		t.Errorf("link notes = %+v", care.LinkNotes)
	}
}

func TestNotesInTheView(t *testing.T) {
	out := view(press(loaded(t, withNotes(), config(&opener{}), 200, 60), "/", "c", "a", "r", "e", "enter"))
	for _, want := range []string{"✎ train 2026.12", "✎ Health: care-dev", "✎ owned by visits-team, degraded", "NOTES ON LINKS", "app ⟶ queue", "40 rps into care-dev-queue"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q:\n%s", want, out)
		}
	}
}

func TestAnnotationsForAServiceAccount(t *testing.T) {
	api := withNotes()
	api.annErr = errors.New("You do not have permission to use annotations as a service account")
	snap := fixtureSnapshotFrom(t, api)
	if snap.AnnotationsErr == nil || len(snap.Annotations) != 0 {
		t.Errorf("annotations = %d, err %v; want none and the reason", len(snap.Annotations), snap.AnnotationsErr)
	}
	out := view(press(loaded(t, api, config(&opener{}), 200, 60), "?"))
	if !strings.Contains(out, "Notes and pinned widgets need a") || !strings.Contains(out, "service account") {
		t.Errorf("help should say why there are no notes:\n%s", out)
	}
}

func TestSlowNote(t *testing.T) {
	api := withNotes()
	api.annotations["care"] = append(api.annotations["care"],
		note(dashboard.SurfaceCanvasLegend, "Environment", "care-dev", "{% for i in (1..100000000) %}x{% endfor %}"))
	start := time.Now()
	g := graphOf(t, api)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("a slow note held the screen for %v", elapsed)
	}
	if notes := g.Env("care-dev").Notes; notes[len(notes)-1] != "(note takes too long to render)" {
		t.Errorf("notes = %q, want the slow one cut off", notes)
	}
	api.stash = append(api.stash, text(env("care-dev"), "counter", "2"))
	start = time.Now()
	g = graphOf(t, api)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("a note that was too slow should not render again with new values; took %v", elapsed)
	}
	if notes := g.Env("care-dev").Notes; notes[len(notes)-1] != "(note takes too long to render)" {
		t.Errorf("notes = %q", notes)
	}
}

func TestPartialStashRead(t *testing.T) {
	api := withStash()
	m := press(loaded(t, api, config(&opener{}), 200, 48), "/", "c", "a", "r", "e", "enter")
	api.failScope = "care-dev"
	out := view(refreshed(t, m))
	if strings.Contains(out, "stash deleted") {
		t.Errorf("a project that could not be read is not deleted:\n%s", out)
	}
	if !strings.Contains(out, "degraded") || !strings.Contains(out, "Load test until 5pm") {
		t.Errorf("a project that could not be read keeps its widgets:\n%s", out)
	}
}
