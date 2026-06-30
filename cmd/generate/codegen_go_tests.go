// Generators for the tests of the server-side Go (codegen_go.go) — one generated test file per
// generated source file. Each emits the gofmt'd output of a templates_go/*_test.go.tmpl.

package main

import "path/filepath"

// generateScoreTest emits game/generated_score_test.go — Score mutators + Equals.
func generateScoreTest(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("score_test.go.tmpl", filepath.Join(destDir, "generated_score_test.go"), buildTemplateData(yamlData))
}

// generateScoreSummaryTest emits game/generated_score_summary_test.go — point math + tiebreaks.
func generateScoreSummaryTest(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("score_summary_test.go.tmpl", filepath.Join(destDir, "generated_score_summary_test.go"), buildTemplateData(yamlData))
}

// generateRankingFieldsTest emits game/generated_ranking_fields_test.go — the ranking-Less cascade.
func generateRankingFieldsTest(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("ranking_fields_test.go.tmpl", filepath.Join(destDir, "generated_ranking_fields_test.go"), buildTemplateData(yamlData))
}
