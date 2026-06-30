// Generate-time template rendering. The committed templates/custom_*.html.tmpl and
// static/js/custom_*.js.tmpl source files are the hand-editable UI defaults; this executes them
// against the view model (see viewmodel.go) to produce the gitignored generated_* files the FMS
// server serves.
//
// Delimiters are [[ ]], not {{ }}. The generated output is itself parsed by the FMS server's own
// html/template engine, so the {{ }} directives in the source (e.g. {{.Position.Title}},
// {{range $i := seq 3}}, {{template "foulButton"}}) must pass through verbatim — using [[ ]] here
// means the generator treats those {{ }} as literal text and only acts on its own [[ ]] directives.

package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"
)

// goTemplates holds the internal Go-emitting templates. Unlike the UI templates/custom_*.tmpl
// (committed, hand-editable game-author surface), these are generator implementation detail —
// embedded, not a customization point — used purely to make the *shape* of the generated Go
// readable in one place instead of buried in strings.Builder calls.
//
//go:embed templates_go/*.tmpl
var goTemplates embed.FS

// renderWebTemplate parses srcPath (a [[ ]]-delimited generate-time template), executes it against
// data, and writes the result to dstPath.
func renderWebTemplate(srcPath, dstPath string, data any) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	tmpl, err := template.New(filepath.Base(srcPath)).Delims("[[", "]]").Funcs(genTemplateFuncs).Parse(string(src))
	if err != nil {
		return err
	}
	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

// renderGoTemplate executes the embedded Go template named name against data, runs the result
// through go/format (so the template itself needn't be perfectly indented — gofmt normalizes it),
// and writes it to dstPath. On a format error it returns the unformatted source for debugging.
func renderGoTemplate(name, dstPath string, data any) error {
	src, err := goTemplates.ReadFile("templates_go/" + name)
	if err != nil {
		return err
	}
	tmpl, err := template.New(name).Delims("[[", "]]").Funcs(genTemplateFuncs).Parse(string(src))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt of generated %s failed: %w\n--- generated source ---\n%s", name, err, buf.String())
	}
	return os.WriteFile(dstPath, formatted, 0644)
}
