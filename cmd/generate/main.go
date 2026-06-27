package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var goIdentRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func main() {
	yamlPath := flag.String("f", "game/game.yaml", "path to game.yaml")
	flag.Parse()

	data, err := os.ReadFile(*yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading game.yaml: %v\n", err)
		os.Exit(1)
	}

	var yamlData GameYAML
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing game.yaml: %v\n", err)
		os.Exit(1)
	}

	// Normalize points
	for i := range yamlData.ScoringCounts {
		sc := &yamlData.ScoringCounts[i]
		if sc.Phase == "auto" && sc.Points > 0 && sc.PointsAuto == 0 {
			sc.PointsAuto = sc.Points
		}
		if sc.Phase == "teleop" && sc.Points > 0 && sc.PointsTeleop == 0 {
			sc.PointsTeleop = sc.Points
		}
	}

	// Validation
	validationErrors := validateGameYAML(&yamlData)

	if len(validationErrors) > 0 {
		fmt.Fprintln(os.Stderr, "Validation errors in game.yaml:")
		for _, errStr := range validationErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", errStr)
		}
		os.Exit(1)
	}

	// Codegen target dir
	destDir := filepath.Dir(*yamlPath) // e.g. "game" if yamlPath is "game/game.yaml"

	if err := generateConstants(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating constants: %v\n", err)
		os.Exit(1)
	}

	if err := generateScore(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Score struct: %v\n", err)
		os.Exit(1)
	}

	if err := generateScoreSummary(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating ScoreSummary: %v\n", err)
		os.Exit(1)
	}

	if err := generateRankingFields(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating RankingFields: %v\n", err)
		os.Exit(1)
	}

	if err := generateScoreTest(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating score test: %v\n", err)
		os.Exit(1)
	}

	if err := appendRPStubs(&yamlData, destDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error appending RP stubs: %v\n", err)
		os.Exit(1)
	}

	templatesDir := filepath.Join(destDir, "../templates")
	staticJsDir := filepath.Join(destDir, "../static/js")
	cmdGenerateDir := filepath.Join(destDir, "../cmd/generate")

	if err := generateScoringPanelTemplate(&yamlData, templatesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating scoring panel template: %v\n", err)
		os.Exit(1)
	}

	if err := generateScoringPanelJS(&yamlData, staticJsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating scoring panel JS: %v\n", err)
		os.Exit(1)
	}

	if err := generateAudienceDisplayTemplate(&yamlData, templatesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating audience display template: %v\n", err)
		os.Exit(1)
	}

	if err := generateAudienceDisplayJS(&yamlData, staticJsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating audience display JS: %v\n", err)
		os.Exit(1)
	}

	if err := generateRefereePanelTemplate(&yamlData, templatesDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating referee panel template: %v\n", err)
		os.Exit(1)
	}

	if err := generateRefereePanelJS(&yamlData, staticJsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating referee panel JS: %v\n", err)
		os.Exit(1)
	}

	if err := generateTemplateTest(&yamlData, cmdGenerateDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating template test: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Code generation complete successfully.")
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func validateGameYAML(yamlData *GameYAML) []string {
	var validationErrors []string

	if yamlData.Game.Name == "" {
		validationErrors = append(validationErrors, "game.name is required")
	}
	if yamlData.Fouls.MinorFoulPoints <= 0 {
		validationErrors = append(validationErrors, "fouls.minor_foul_points must be > 0")
	}
	if yamlData.Fouls.MajorFoulPoints <= 0 {
		validationErrors = append(validationErrors, "fouls.major_foul_points must be > 0")
	}

	seenIDs := make(map[string]bool)
	checkDup := func(id string, context string) {
		if id == "" {
			return
		}
		if seenIDs[id] {
			validationErrors = append(validationErrors, fmt.Sprintf("duplicate id: '%s' in %s", id, context))
		}
		seenIDs[id] = true
	}

	// Game pieces
	gamePieces := make(map[string]bool)
	for i, gp := range yamlData.GamePieces {
		if gp.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("game_pieces[%d]: id is required", i))
		} else if !goIdentRegexp.MatchString(gp.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("game_pieces[%d]: id '%s' must be a valid Go identifier (letters, digits, underscores; cannot start with a digit)", i, gp.ID))
		} else {
			checkDup(gp.ID, "game_pieces")
			gamePieces[gp.ID] = true
		}
	}

	// scoring_counts
	for i, sc := range yamlData.ScoringCounts {
		if sc.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d]: id is required", i))
			continue
		}
		if !goIdentRegexp.MatchString(sc.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d]: id '%s' must be a valid Go identifier (letters, digits, underscores; cannot start with a digit)", i, sc.ID))
			continue
		}
		checkDup(sc.ID, "scoring_counts")

		if sc.GamePiece != "" && !gamePieces[sc.GamePiece] {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: unknown game_piece '%s'", i, sc.ID, sc.GamePiece))
		}

		if sc.Phase != "auto" && sc.Phase != "teleop" && sc.Phase != "both" {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: unknown phase '%s'", i, sc.ID, sc.Phase))
		} else {
			if sc.Phase == "both" && (sc.PointsAuto <= 0 || sc.PointsTeleop <= 0) {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: phase 'both' requires points_auto and points_teleop", i, sc.ID))
			}
			if sc.Phase == "auto" && sc.PointsAuto <= 0 && sc.Points <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: phase 'auto' requires points or points_auto", i, sc.ID))
			}
			if sc.Phase == "teleop" && sc.PointsTeleop <= 0 && sc.Points <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: phase 'teleop' requires points or points_teleop", i, sc.ID))
			}
		}
	}

	// endgame_counts
	for i, ec := range yamlData.EndgameCounts {
		if ec.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("endgame_counts[%d]: id is required", i))
			continue
		}
		if !goIdentRegexp.MatchString(ec.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("endgame_counts[%d]: id '%s' must be a valid Go identifier (letters, digits, underscores; cannot start with a digit)", i, ec.ID))
			continue
		}
		checkDup(ec.ID, "endgame_counts")

		if ec.Points <= 0 && ec.PointsAuto <= 0 && ec.PointsTeleop <= 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("endgame_counts[%d].%s: points must be > 0", i, ec.ID))
		}
	}

	// statuses
	for i, status := range yamlData.Statuses {
		if status.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d]: id is required", i))
			continue
		}
		if !goIdentRegexp.MatchString(status.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d]: id '%s' must be a valid Go identifier (letters, digits, underscores; cannot start with a digit)", i, status.ID))
			continue
		}
		checkDup(status.ID, "statuses")

		if status.Phase != "auto" && status.Phase != "endgame" {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: unknown phase '%s'", i, status.ID, status.Phase))
		}

		if len(status.Values) > 0 {
			if len(status.Values) < 2 {
				validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: enum status requires at least 2 values", i, status.ID))
			}
			statusVals := make(map[string]bool)
			for j, val := range status.Values {
				if val.ID == "" {
					validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s values[%d]: id is required", i, status.ID, j))
				} else {
					if statusVals[val.ID] {
						validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: duplicate value id '%s'", i, status.ID, val.ID))
					}
					statusVals[val.ID] = true
				}
			}
		} else {
			if status.Points <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: points must be > 0 for bool status", i, status.ID))
			}
		}
	}

	// RPs
	for i, rp := range yamlData.RPs {
		if rp.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_points[%d]: id is required", i))
		} else {
			checkDup(rp.ID, "ranking_points")
		}

		if rp.LogicFunc == "" || !goIdentRegexp.MatchString(rp.LogicFunc) {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_points[%d].logic_func: '%s' is not a valid Go identifier", i, rp.LogicFunc))
		}
	}

	// Build map of valid metrics
	validElements := map[string]bool{
		"auto_points":    true,
		"teleop_points":  true,
		"endgame_points": true,
		"total_points":   true,
	}
	for _, sc := range yamlData.ScoringCounts {
		validElements[sc.ID] = true
	}
	for _, ec := range yamlData.EndgameCounts {
		validElements[ec.ID] = true
	}
	for _, status := range yamlData.Statuses {
		validElements[status.ID] = true
	}

	// ranking_tiebreakers
	for i, tb := range yamlData.RankingTiebreakers {
		if !validElements[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_tiebreakers[%d]: unknown metric '%s'", i, tb.Metric))
		}
	}

	// playoff_tiebreakers
	for i, tb := range yamlData.PlayoffTiebreakers {
		if !validElements[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("playoff_tiebreakers[%d]: unknown metric '%s'", i, tb.Metric))
		}
	}

	return validationErrors
}
