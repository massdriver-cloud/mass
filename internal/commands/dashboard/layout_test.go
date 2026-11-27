package dashboard_test

import (
	"maps"
	"math"
	"testing"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func TestLayout(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	tests := []struct {
		name string
		opts dashboard.LayoutOptions
		w, h int
	}{
		{name: "free", w: 120, h: 40},
		{name: "grouped by project", opts: dashboard.LayoutOptions{GroupByProject: true}, w: 120, h: 40},
		{name: "producers on top", opts: dashboard.LayoutOptions{ProducersOnTop: true}, w: 120, h: 40},
		{name: "cramped", w: 30, h: 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos := dashboard.Layout(g, tc.w, tc.h, tc.opts)
			if len(pos) != len(g.Envs) {
				t.Fatalf("placed %d of %d environments", len(pos), len(g.Envs))
			}
			for id, p := range pos {
				if p.X < 0 || p.Y < 0 || p.X >= float64(tc.w*2) || p.Y >= float64(tc.h*4) || math.IsNaN(p.X) {
					t.Errorf("%s at %+v is off the %dx%d canvas", id, p, tc.w, tc.h)
				}
			}
			if again := dashboard.Layout(g, tc.w, tc.h, tc.opts); !maps.Equal(pos, again) {
				t.Error("the same graph laid out twice landed in different places")
			}
		})
	}
}

func TestLayoutProducersOnTop(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.Layout(g, 160, 48, dashboard.LayoutOptions{ProducersOnTop: true})
	if !(pos["platform-dev"].Y < pos["care-dev"].Y && pos["care-dev"].Y < pos["billing-dev"].Y) {
		t.Errorf("platform %v, care %v, billing %v: producers should sit above consumers",
			pos["platform-dev"], pos["care-dev"], pos["billing-dev"])
	}
}

func TestLayoutKeepsLabelsApart(t *testing.T) {
	g := dashboard.BuildGraph(fixtureSnapshot(t))
	pos := dashboard.Layout(g, 160, 48, dashboard.LayoutOptions{})
	for _, a := range g.Envs {
		for _, b := range g.Envs {
			if a.ID >= b.ID {
				continue
			}
			pa, pb := pos[a.ID], pos[b.ID]
			if math.Abs(pa.X-pb.X)/2 < 6 && math.Abs(pa.Y-pb.Y)/4 < 4 {
				t.Errorf("%s %v and %s %v overlap", a.ID, pa, b.ID, pb)
			}
		}
	}
}

func TestLayoutEmpty(t *testing.T) {
	if pos := dashboard.Layout(dashboard.BuildGraph(&dashboard.Snapshot{}), 80, 24, dashboard.LayoutOptions{}); len(pos) != 0 {
		t.Errorf("layout = %v, want empty", pos)
	}
}
