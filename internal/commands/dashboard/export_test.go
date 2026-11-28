package dashboard_test

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/massdriver-cloud/mass/internal/commands/dashboard"
)

func validXML(t *testing.T, doc string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(doc))
	for {
		_, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("not valid XML: %v", err)
		}
	}
}

func TestExportSVG(t *testing.T) {
	const w, h = 140, 40
	snap := fixtureSnapshot(t)
	snap.Environments[0].Name = `Dev <"prod"> & 'friends'`
	g := dashboard.BuildGraph(snap)
	for _, theme := range dashboard.Themes() {
		for _, palette := range dashboard.Palettes() {
			t.Run(theme+"/"+palette, func(t *testing.T) {
				if err := dashboard.UsePalette(palette); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = dashboard.UsePalette(dashboard.DefaultPalette) })
				scene := dashboard.Scene{
					Graph: g, Pos: dashboard.LayoutTheme(theme, g, w, h, dashboard.LayoutOptions{Center: "care-dev"}),
					Selected: "platform-dev", Blast: g.BlastRadius("platform-dev-db"), Theme: theme,
					ShowApps: true, ShowCost: true, Now: fixedNow(),
				}
				out := dashboard.ExportSVG(scene, w, h, "acme & co · Care / Development")
				validXML(t, out)
				for _, env := range g.Envs {
					if !strings.Contains(out, xmlText(env.Name)) {
						t.Errorf("export is missing %q", env.Name)
					}
				}
				for _, want := range []string{"Dev &lt;&#34;prod&#34;&gt; &amp; &#39;friends&#39;", "acme &amp; co", "▲1", "monospace"} {
					if !strings.Contains(out, want) {
						t.Errorf("export is missing %q", want)
					}
				}
				if strings.Contains(out, `<"prod">`) {
					t.Error("a name should be escaped")
				}
			})
		}
	}
}

func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func TestExportName(t *testing.T) {
	at := time.Date(2026, 11, 26, 9, 5, 0, 0, time.UTC)
	tests := []struct{ org, want string }{
		{"acme", "constellation-acme-20261126-0905.svg"},
		{"../a b", "constellation----a-b-20261126-0905.svg"},
	}
	for _, tc := range tests {
		if got := dashboard.ExportName(tc.org, at); got != tc.want {
			t.Errorf("ExportName(%q) = %q, want %q", tc.org, got, tc.want)
		}
	}
}

type fileWriter struct {
	names []string
	data  []string
	err   error
}

func (f *fileWriter) write(name string, data []byte) error {
	f.names, f.data = append(f.names, name), append(f.data, string(data))
	return f.err
}

func TestModelExports(t *testing.T) {
	tests := []struct {
		name      string
		keys      []string
		err       error
		wantWrite bool
		wantToast string
	}{
		{name: "constellation", keys: []string{"e"}, wantWrite: true, wantToast: "saved the map to constellation-acme-20261126-0000.svg"},
		{name: "metro", keys: []string{"v", "e"}, wantWrite: true, wantToast: "saved the map to"},
		{name: "write fails", keys: []string{"e"}, err: errors.New("read-only"), wantWrite: true, wantToast: "could not save the map: read-only"},
		{name: "other tab", keys: []string{"2", "e"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fw := &fileWriter{err: tc.err}
			cfg := config(&opener{})
			cfg.WriteFile = fw.write
			m, cmd := pressCmd(loaded(t, fixture(), cfg, 200, 48), tc.keys...)
			if !tc.wantWrite {
				if cmd != nil {
					t.Error("e off the constellation should not export")
				}
				return
			}
			if cmd == nil {
				t.Fatal("e should export the map")
			}
			next, _ := m.Update(cmd())
			if len(fw.names) != 1 || fw.names[0] != "constellation-acme-20261126-0000.svg" {
				t.Fatalf("wrote %v, want one constellation-acme-20261126-0000.svg", fw.names)
			}
			validXML(t, fw.data[0])
			if !strings.Contains(fw.data[0], "acme · Care / Development · 2026-11-26 00:00 UTC") {
				t.Error("the export should be titled with the org, focus, and time")
			}
			if out := view(next.(dashboard.Model)); !strings.Contains(out, tc.wantToast) { //nolint:errcheck,forcetypeassert // Update always returns a Model
				t.Errorf("footer is missing %q:\n%s", tc.wantToast, out)
			}
		})
	}
}
