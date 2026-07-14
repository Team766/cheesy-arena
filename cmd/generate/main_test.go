package main

import (
	"github.com/stretchr/testify/assert"
	"go/ast"
	"go/parser"
	"go/token"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTemplates(t *testing.T) {
	paths := []string{
		"../../game/custom_game.yaml",
		"../../game/examples/high_seas_havoc.yaml",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			data, err := os.ReadFile(p)
			assert.Nil(t, err)

			var yamlData GameYAML
			err = yaml.Unmarshal(data, &yamlData)
			assert.Nil(t, err)

			validationErrors := validateGameYAML(&yamlData)
			assert.Empty(t, validationErrors)
		})
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		modify        func(*GameYAML)
		expectedError string
	}{
		{
			name: "missing game name",
			modify: func(y *GameYAML) {
				y.Game.Name = ""
			},
			expectedError: "game.name is required",
		},
		{
			name: "invalid minor foul points",
			modify: func(y *GameYAML) {
				y.Fouls.MinorFoulPoints = 0
			},
			expectedError: "fouls.minor_foul_points must be > 0",
		},
		{
			name: "invalid major foul points",
			modify: func(y *GameYAML) {
				y.Fouls.MajorFoulPoints = -1
			},
			expectedError: "fouls.major_foul_points must be > 0",
		},
		{
			name: "missing scoring count id",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].ID = ""
			},
			expectedError: "scoring_counts[0]: id is required",
		},
		{
			name: "bad scoring count phase",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases[0].Phase = "invalid_phase"
			},
			expectedError: "unknown phase 'invalid_phase'",
		},
		{
			name: "scoring count with no phases",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases = nil
			},
			expectedError: "at least one phase is required",
		},
		{
			name: "scoring count with duplicate phase",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases = []PhasePoints{
					{Phase: "auto", Points: 5},
					{Phase: "auto", Points: 3},
				}
			},
			expectedError: "duplicate phase 'auto'",
		},
		{
			name: "scoring count phase with non-positive points",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases = []PhasePoints{{Phase: "auto", Points: 0}}
			},
			expectedError: "points must be > 0",
		},
		{
			name: "scoring count in both teleop and endgame rejected",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases = []PhasePoints{{Phase: "teleop", Points: 2}, {Phase: "endgame", Points: 3}}
			},
			expectedError: "cannot be scored in both teleop and endgame",
		},
		{
			name: "unknown scoring_group reference",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].ScoringGroup = "nonexistent"
			},
			expectedError: "unknown scoring_group 'nonexistent'",
		},
		{
			name: "missing game_piece rejected",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].GamePiece = ""
			},
			expectedError: "game_piece is required",
		},
		{
			name: "enum status with too few values",
			modify: func(y *GameYAML) {
				y.Statuses = []Status{
					{
						ID:     "bad_status",
						Phases: []PhasePoints{{Phase: "auto"}},
						Values: []StatusValue{
							{ID: "one", DisplayName: "One"},
						},
					},
				}
			},
			expectedError: "enum status requires at least 2 values",
		},
		{
			name: "status with teleop phase rejected",
			modify: func(y *GameYAML) {
				y.Statuses[0].Phases = []PhasePoints{{Phase: "teleop", Points: 3}}
			},
			expectedError: "only auto and endgame are supported for statuses",
		},
		{
			name: "status with more than one phase rejected",
			modify: func(y *GameYAML) {
				y.Statuses[0].Phases = []PhasePoints{{Phase: "auto", Points: 3}, {Phase: "endgame", Points: 3}}
			},
			expectedError: "exactly one phase is required",
		},
		{
			name: "unknown tiebreaker metric",
			modify: func(y *GameYAML) {
				y.RankingTiebreakers = append(y.RankingTiebreakers, Tiebreaker{Metric: "nonexistent"})
			},
			expectedError: "unknown metric 'nonexistent'",
		},
		{
			name: "duplicate id across sections",
			modify: func(y *GameYAML) {
				// Duplicate leave in scoring counts
				y.ScoringCounts = append(y.ScoringCounts, ScoringCount{ID: "leave", GamePiece: y.GamePieces[0].ID, Phases: []PhasePoints{{Phase: "auto", Points: 5}}})
			},
			expectedError: "duplicate id: 'leave'",
		},
		{
			name: "id collides with a built-in summary field",
			modify: func(y *GameYAML) {
				// "match" -> MatchPoints, which already exists as a built-in ScoreSummary field.
				y.Statuses[0].ID = "match"
			},
			expectedError: "collides with a built-in field",
		},
		{
			name: "two ids generate the same summary field",
			modify: func(y *GameYAML) {
				// "Structure1" CamelCases to the same field as scoring_group "structure1"
				// (-> Structure1Points), yet is a distinct raw id, so the dup-id check misses it.
				y.Statuses = append(y.Statuses, Status{ID: "Structure1", Phases: []PhasePoints{{Phase: "auto", Points: 3}}})
			},
			expectedError: "collides with scoring group 'structure1'",
		},
		{
			name: "duplicate ranking tiebreaker metric",
			modify: func(y *GameYAML) {
				// The default already lists total_points; a second entry is a duplicate that would
				// emit a duplicate RankingFields struct field.
				y.RankingTiebreakers = append(y.RankingTiebreakers, Tiebreaker{Metric: "total_points"})
			},
			expectedError: "duplicate metric 'total_points'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with a clean copy of default template
			data, err := os.ReadFile("../../game/custom_game.yaml")
			assert.Nil(t, err)

			var yamlData GameYAML
			err = yaml.Unmarshal(data, &yamlData)
			assert.Nil(t, err)

			tt.modify(&yamlData)

			validationErrors := validateGameYAML(&yamlData)
			assert.NotEmpty(t, validationErrors)

			found := false
			for _, errStr := range validationErrors {
				if assert.Contains(t, errStr, tt.expectedError) {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected error containing: %q, got: %v", tt.expectedError, validationErrors)
		})
	}
}

func TestValidateCustomScoringLogic(t *testing.T) {
	// Base config: the default template, whose ranking_points name ComputeAutonRP/ScoringRP/EndgameRP
	// and whose Score exposes AutoStructure1Level1Count etc.
	data, err := os.ReadFile("../../game/custom_game.yaml")
	assert.Nil(t, err)
	var base GameYAML
	assert.Nil(t, yaml.Unmarshal(data, &base))

	writeLogic := func(t *testing.T, content string) string {
		path := filepath.Join(t.TempDir(), "custom_scoring_logic.go")
		assert.Nil(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	// A logic file with all three declared logic funcs and the given body for ComputeAutonRP.
	logicWith := func(autonBody string) string {
		return "package game\n" +
			"func ComputeAutonRP(score, opponentScore Score, summary ScoreSummary) bool {\n" + autonBody + "\n}\n" +
			"func ComputeScoringRP(score, opponentScore Score, summary ScoreSummary) bool { return false }\n" +
			"func ComputeEndgameRP(score, opponentScore Score, summary ScoreSummary) bool { return false }\n"
	}

	t.Run("valid logic matching default config", func(t *testing.T) {
		logic, err := os.ReadFile("../../game/custom_scoring_logic.go")
		assert.Nil(t, err)
		assert.Empty(t, validateCustomScoringLogic(&base, writeLogic(t, string(logic))))
	})

	t.Run("no false positives on method calls and non-param selectors", func(t *testing.T) {
		path := writeLogic(t, logicWith(
			"\tfor _, foul := range score.Fouls {\n\t\t_ = foul.PointValue()\n\t}\n\treturn summary.AutoPoints >= 9"))
		assert.Empty(t, validateCustomScoringLogic(&base, path))
	})

	t.Run("obsolete Score field with suggestion", func(t *testing.T) {
		// A near-miss typo of a real field should be flagged and suggest the real field.
		path := writeLogic(t, logicWith("\treturn score.AutoStructure1Level1Kount > 2"))
		errs := validateCustomScoringLogic(&base, path)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "AutoStructure1Level1Kount")
		assert.Contains(t, errs[0], "did you mean 'AutoStructure1Level1Count'")
	})

	t.Run("unknown ScoreSummary field", func(t *testing.T) {
		path := writeLogic(t, logicWith("\treturn summary.BogusPoints > 0"))
		errs := validateCustomScoringLogic(&base, path)
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "summary.BogusPoints")
		assert.Contains(t, errs[0], "is not a field generated")
	})

	t.Run("missing logic func", func(t *testing.T) {
		content := "package game\n" +
			"func ComputeAutonRP(score, opponentScore Score, summary ScoreSummary) bool { return false }\n" +
			"func ComputeScoringRP(score, opponentScore Score, summary ScoreSummary) bool { return false }\n"
		errs := validateCustomScoringLogic(&base, writeLogic(t, content))
		assert.Len(t, errs, 1)
		assert.Contains(t, errs[0], "ComputeEndgameRP")
		assert.Contains(t, errs[0], "no such function")
	})

	t.Run("missing file with no ranking points is fine", func(t *testing.T) {
		noRPs := base
		noRPs.RPs = nil
		assert.Empty(t, validateCustomScoringLogic(&noRPs, filepath.Join(t.TempDir(), "does_not_exist.go")))
	})
}

// TestGeneratedFieldSetsMatchTemplates guards the hand-maintained base-field lists in
// generatedFieldSets (validate_logic.go) against drifting from what score.go.tmpl /
// score_summary.go.tmpl actually emit. If a base field is added to a template but not to
// generatedFieldSets, the validator would falsely reject a valid custom_scoring_logic.go and halt
// generation; this catches that. Runs against the generated structs for the default config; skipped
// on a fresh checkout where `go generate` hasn't produced them yet.
func TestGeneratedFieldSetsMatchTemplates(t *testing.T) {
	data, err := os.ReadFile("../../game/custom_game.yaml")
	assert.Nil(t, err)
	var y GameYAML
	assert.Nil(t, yaml.Unmarshal(data, &y))
	scoreFields, summaryFields := generatedFieldSets(&y)

	check := func(genPath, structName string, allowed map[string]bool) {
		src, err := os.ReadFile(genPath)
		if err != nil {
			t.Skipf("%s not present; run `go generate ./...` first (%v)", genPath, err)
		}
		for _, field := range structFieldNames(t, src, structName) {
			assert.Truef(t, allowed[field],
				"generated %s.%s is missing from generatedFieldSets in validate_logic.go — the validator will falsely reject it",
				structName, field)
		}
	}
	check("../../game/generated_score.go", "Score", scoreFields)
	check("../../game/generated_score_summary.go", "ScoreSummary", summaryFields)
}

// structFieldNames returns the declared field names of the named struct in Go source src.
func structFieldNames(t *testing.T, src []byte, structName string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, 0)
	assert.Nil(t, err)
	var names []string
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != structName {
			return true
		}
		if st, ok := ts.Type.(*ast.StructType); ok {
			for _, f := range st.Fields.List {
				for _, nm := range f.Names {
					names = append(names, nm.Name)
				}
			}
		}
		return false
	})
	return names
}
