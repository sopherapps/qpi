package drivers

import (
	"strings"
	"text/template"
)

// mustRender executes a setup template against data and returns the result.
// It panics on error, which — like the template.Must the templates are built
// with — can only happen if the template itself is malformed, never on the
// plain-string data passed here.
func mustRender(t *template.Template, data snippetData) string {
	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// shellQuote wraps a value in single quotes for safe interpolation into the
// setup snippets, escaping any embedded single quotes. A driver's name and the
// bracketed pip extra are spliced into a shell command the operator later
// pastes and runs, so they must not be able to break out of it.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
