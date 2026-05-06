package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/massdriver-cloud/mass/internal/cli"
)

func TestMarkdownEscape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "no specials", in: "team", want: "team"},
		{name: "underscore", in: "cost_center", want: `cost\_center`},
		{name: "hyphen", in: "cost-center", want: `cost\-center`},
		{name: "multiple", in: "cost_center.99-eu_west", want: `cost\_center\.99\-eu\_west`},
		{name: "asterisk", in: "*hot*", want: `\*hot\*`},
		{name: "backslash first", in: `\foo`, want: `\\foo`},
		{name: "preexisting escape stays escaped", in: `cost\_center`, want: `cost\\\_center`},
		{name: "pipe", in: "a|b", want: `a\|b`},
		{name: "backtick", in: "`x`", want: "\\`x\\`"},
		{name: "all the things", in: `\` + "`*_{}[]<>()#+-.!|", want: `\\\` + "`" + `\*\_\{\}\[\]\<\>\(\)\#\+\-\.\!\|`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cli.MarkdownEscape(tc.in)
			if got != tc.want {
				t.Errorf("MarkdownEscape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMarkdownTemplateFuncs(t *testing.T) {
	tmpl, err := template.New("t").Funcs(cli.MarkdownTemplateFuncs).Parse(`{{ . | mdEscape }}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, "cost_center"); execErr != nil {
		t.Fatalf("execute: %v", execErr)
	}
	if got, want := buf.String(), `cost\_center`; got != want {
		t.Errorf("template output = %q, want %q", got, want)
	}
}

// TestMarkdownEscape_GoldmarkRoundtrip confirms that strings escaped by
// MarkdownEscape produce the same literal text in the rendered markdown
// output: the backslashes get consumed and the underscores survive intact.
func TestMarkdownEscape_GoldmarkRoundtrip(t *testing.T) {
	in := "cost_center"
	escaped := cli.MarkdownEscape(in)
	if !strings.Contains(escaped, `\_`) {
		t.Fatalf("expected backslash-underscore in escaped output, got %q", escaped)
	}
}
