package cli

import (
	"strings"
	"text/template"
)

// markdownEscapeReplacer backslash-escapes every character goldmark treats
// as a markdown punctuation escape. The leading `\` substitution must come
// first so we don't re-escape backslashes we just inserted.
var markdownEscapeReplacer = strings.NewReplacer(
	`\`, `\\`,
	"`", "\\`",
	"*", `\*`,
	"_", `\_`,
	"{", `\{`,
	"}", `\}`,
	"[", `\[`,
	"]", `\]`,
	"<", `\<`,
	">", `\>`,
	"(", `\(`,
	")", `\)`,
	"#", `\#`,
	"+", `\+`,
	"-", `\-`,
	".", `\.`,
	"!", `\!`,
	"|", `\|`,
)

// MarkdownEscape escapes markdown punctuation in user-supplied strings
// before they land in a template that gets fed to glamour. Without this,
// characters like `_` in `cost_center` open emphasis spans and the renderer
// mangles the output (e.g. `cost_center` displays as `cost_****center`).
func MarkdownEscape(s string) string {
	return markdownEscapeReplacer.Replace(s)
}

// MarkdownTemplateFuncs are the helper funcs available in every glamour-fed
// markdown template. Pass to `template.New(...).Funcs(...)`.
var MarkdownTemplateFuncs = template.FuncMap{
	"mdEscape": MarkdownEscape,
}
