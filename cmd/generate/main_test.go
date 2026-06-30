package main

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
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
