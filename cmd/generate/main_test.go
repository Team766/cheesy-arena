package main

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestValidateTemplates(t *testing.T) {
	paths := []string{
		"../../game/game.yaml",
		"../../examples/mayhem_2025.yaml",
		"../../examples/rebuilt_2026.yaml",
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
			name: "bad element phase",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phase = "invalid_phase"
			},
			expectedError: "unknown phase 'invalid_phase'",
		},
		{
			name: "both phase missing points",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phase = "both"
				y.ScoringCounts[0].PointsAuto = 0
				y.ScoringCounts[0].PointsTeleop = 0
			},
			expectedError: "requires points_auto and points_teleop",
		},
		{
			name: "enum status with too few values",
			modify: func(y *GameYAML) {
				y.Statuses = []Status{
					{
						ID:    "bad_status",
						Phase: "auto",
						Values: []StatusValue{
							{ID: "one", DisplayName: "One"},
						},
					},
				}
			},
			expectedError: "enum status requires at least 2 values",
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
				y.ScoringCounts = append(y.ScoringCounts, Element{ID: "leave", Phase: "auto", Points: 5})
			},
			expectedError: "duplicate id: 'leave'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with a clean copy of default template
			data, err := os.ReadFile("../../game/game.yaml")
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
