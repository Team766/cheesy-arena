// Generator for the test of the web UI surfaces (codegen_web.go): emits
// cmd/generate/generated_template_test.go, which asserts the rendered templates parse and contain
// the expected per-element markup.

package main

import "path/filepath"

func generateTemplateTest(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("template_test.go.tmpl", filepath.Join(destDir, "generated_template_test.go"), buildTemplateData(yamlData))
}
