// Code generators for the server-side Go: game/generated_{constants,score,score_summary,
// ranking_fields}.go. Each emits the gofmt'd output of a templates_go/*.go.tmpl executed against
// the view model (see viewmodel.go). Companion to codegen_web.go (HTML/JS) and codegen_tests.go.

package main

import "path/filepath"

var phaseFieldPrefix = map[string]string{"auto": "Auto", "teleop": "Teleop", "endgame": "Endgame"}
var phaseConstName = map[string]string{"auto": "PhaseAuto", "teleop": "PhaseTeleop", "endgame": "PhaseEndgame"}

// generateConstants emits game/generated_constants.go — game-wide metadata, mode flags, foul points.
func generateConstants(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("constants.go.tmpl", filepath.Join(destDir, "generated_constants.go"), buildTemplateData(yamlData))
}

// generateScore emits game/generated_score.go — the Score struct, Equals, the Phase enum, and the
// Adjust/Set/Cycle scoring methods + dispatchers.
func generateScore(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("score.go.tmpl", filepath.Join(destDir, "generated_score.go"), buildTemplateData(yamlData))
}

// generateScoreSummary emits game/generated_score_summary.go — ScoreSummary, per-phase point
// accumulation, and the DetermineMatchStatus tiebreak cascade.
func generateScoreSummary(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("score_summary.go.tmpl", filepath.Join(destDir, "generated_score_summary.go"), buildTemplateData(yamlData))
}

// generateRankingFields emits game/generated_ranking_fields.go — RankingFields, AddScoreSummary,
// and the Less tiebreaker cascade.
func generateRankingFields(yamlData *GameYAML, destDir string) error {
	return renderGoTemplate("ranking_fields.go.tmpl", filepath.Join(destDir, "generated_ranking_fields.go"), buildTemplateData(yamlData))
}
