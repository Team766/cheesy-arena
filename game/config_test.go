package game

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testGameYAML is a small, self-contained, valid config that the validation tests mutate. Using it
// instead of the shipped game/custom_game.yaml keeps these unit tests independent of whichever game
// is configured — a user who swaps in their own YAML doesn't break the engine's own tests.
func testGameYAML() *GameYAML {
	return &GameYAML{
		Game:          GameInfo{Name: "Test Game"},
		Fouls:         FoulConfig{MinorFoulPoints: 5, MajorFoulPoints: 15},
		GamePieces:    []GamePiece{{ID: "cube", DisplayName: "Cube"}},
		ScoringGroups: []ScoringGroup{{ID: "rack", DisplayName: "Rack"}},
		ScoringCounts: []ScoringCount{
			{
				ID: "rack_low", DisplayName: "Rack Low", GamePiece: "cube", ScoringGroup: "rack",
				Phases: []PhasePoints{{Phase: "auto", Points: 3}, {Phase: "teleop", Points: 2}},
			},
			{
				ID: "rack_high", DisplayName: "Rack High", GamePiece: "cube", ScoringGroup: "rack",
				Phases: []PhasePoints{{Phase: "teleop", Points: 5}},
			},
			{
				ID: "solo", DisplayName: "Solo", GamePiece: "cube",
				Phases: []PhasePoints{{Phase: "endgame", Points: 4}},
			},
		},
		Statuses: []Status{
			{ID: "park", DisplayName: "Park", Phases: []PhasePoints{{Phase: "endgame", Points: 2}}},
			{
				ID: "climb", DisplayName: "Climb", Phases: []PhasePoints{{Phase: "endgame"}},
				Values: []StatusValue{
					{ID: "none", DisplayName: "None", Points: 0},
					{ID: "high", DisplayName: "High", Points: 5},
				},
			},
		},
		RPs:                []RankingPoint{{ID: "auto_rp", DisplayName: "Auto RP", LogicFunc: "ComputeAutoRp"}},
		RankingTiebreakers: []Tiebreaker{{Metric: "total_points"}, {Metric: "auto_points"}},
		PlayoffTiebreakers: []Tiebreaker{{Metric: "auto_points"}, {Metric: "rack"}},
	}
}

func TestValidateGameYAMLAcceptsTheFixture(t *testing.T) {
	assert.Empty(t, ValidateGameYAML(testGameYAML()))
}

type validationCase struct {
	name          string
	modify        func(*GameYAML)
	expectedError string
}

func TestValidationErrors(t *testing.T) {
	tests := []validationCase{
		{
			name:          "missing game name",
			modify:        func(y *GameYAML) { y.Game.Name = "" },
			expectedError: "game.name is required",
		},
		{
			name:          "invalid minor foul points",
			modify:        func(y *GameYAML) { y.Fouls.MinorFoulPoints = 0 },
			expectedError: "fouls.minor_foul_points must be > 0",
		},
		{
			name:          "invalid major foul points",
			modify:        func(y *GameYAML) { y.Fouls.MajorFoulPoints = -1 },
			expectedError: "fouls.major_foul_points must be > 0",
		},
		{
			name:          "missing game piece id",
			modify:        func(y *GameYAML) { y.GamePieces[0].ID = "" },
			expectedError: "game_pieces[0]: id is required",
		},
		{
			name:          "game piece id is not an identifier",
			modify:        func(y *GameYAML) { y.GamePieces[0].ID = "not a name" },
			expectedError: "game_pieces[0]: id 'not a name' must be a valid identifier",
		},
		{
			name:          "duplicate game piece id",
			modify:        func(y *GameYAML) { y.GamePieces = append(y.GamePieces, GamePiece{ID: "cube"}) },
			expectedError: "duplicate id: 'cube' in game_pieces",
		},
		{
			name:          "missing scoring group id",
			modify:        func(y *GameYAML) { y.ScoringGroups[0].ID = "" },
			expectedError: "scoring_groups[0]: id is required",
		},
		{
			name:          "scoring group id is not an identifier",
			modify:        func(y *GameYAML) { y.ScoringGroups[0].ID = "rack-1" },
			expectedError: "scoring_groups[0]: id 'rack-1' must be a valid identifier",
		},
		{
			name:          "missing scoring count id",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].ID = "" },
			expectedError: "scoring_counts[0]: id is required",
		},
		{
			name:          "scoring count id is not an identifier",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].ID = "9lives" },
			expectedError: "scoring_counts[0]: id '9lives' must be a valid identifier",
		},
		{
			name:          "missing game_piece rejected",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].GamePiece = "" },
			expectedError: "game_piece is required",
		},
		{
			name:          "unknown game_piece reference",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].GamePiece = "nonexistent" },
			expectedError: "unknown game_piece 'nonexistent'",
		},
		{
			name:          "unknown scoring_group reference",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].ScoringGroup = "nonexistent" },
			expectedError: "unknown scoring_group 'nonexistent'",
		},
		{
			name:          "scoring count with an invalid scorer hint",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].Scorer = "middle" },
			expectedError: "scorer must be 'near' or 'far', got 'middle'",
		},
		{
			name:          "status with an invalid scorer hint",
			modify:        func(y *GameYAML) { y.Statuses[0].Scorer = "both" },
			expectedError: "scorer must be 'near' or 'far', got 'both'",
		},
		{
			name:          "scoring count with no phases",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].Phases = nil },
			expectedError: "at least one phase is required",
		},
		{
			name:          "bad scoring count phase",
			modify:        func(y *GameYAML) { y.ScoringCounts[0].Phases[0].Phase = "invalid_phase" },
			expectedError: "unknown phase 'invalid_phase'",
		},
		{
			name: "scoring count with duplicate phase",
			modify: func(y *GameYAML) {
				y.ScoringCounts[0].Phases = []PhasePoints{{Phase: "auto", Points: 5}, {Phase: "auto", Points: 3}}
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
			name:          "missing status id",
			modify:        func(y *GameYAML) { y.Statuses[0].ID = "" },
			expectedError: "statuses[0]: id is required",
		},
		{
			name:          "status id is not an identifier",
			modify:        func(y *GameYAML) { y.Statuses[0].ID = "park!" },
			expectedError: "statuses[0]: id 'park!' must be a valid identifier",
		},
		{
			name: "status with more than one phase rejected",
			modify: func(y *GameYAML) {
				y.Statuses[0].Phases = []PhasePoints{{Phase: "auto", Points: 3}, {Phase: "endgame", Points: 3}}
			},
			expectedError: "exactly one phase is required",
		},
		{
			name:          "status with no phase rejected",
			modify:        func(y *GameYAML) { y.Statuses[0].Phases = nil },
			expectedError: "exactly one phase is required",
		},
		{
			name: "status with teleop phase rejected",
			modify: func(y *GameYAML) {
				y.Statuses[0].Phases = []PhasePoints{{Phase: "teleop", Points: 3}}
			},
			expectedError: "only auto and endgame are supported for statuses",
		},
		{
			name: "bool status with non-positive points",
			modify: func(y *GameYAML) {
				y.Statuses[0].Phases = []PhasePoints{{Phase: "endgame", Points: 0}}
			},
			expectedError: "points must be > 0 for bool status",
		},
		{
			name: "enum status with too few values",
			modify: func(y *GameYAML) {
				y.Statuses[1].Values = []StatusValue{{ID: "one", DisplayName: "One"}}
			},
			expectedError: "enum status requires at least 2 values",
		},
		{
			name: "enum status first value scores points",
			modify: func(y *GameYAML) {
				y.Statuses[1].Values[0].Points = 2
			},
			expectedError: "must have points: 0",
		},
		{
			name: "enum status value missing an id",
			modify: func(y *GameYAML) {
				y.Statuses[1].Values[1].ID = ""
			},
			expectedError: "values[1]: id is required",
		},
		{
			name: "enum status with duplicate value ids",
			modify: func(y *GameYAML) {
				y.Statuses[1].Values = append(y.Statuses[1].Values, StatusValue{ID: "none", Points: 3})
			},
			expectedError: "duplicate value id 'none'",
		},
		{
			name:          "missing ranking point id",
			modify:        func(y *GameYAML) { y.RPs[0].ID = "" },
			expectedError: "ranking_points[0]: id is required",
		},
		{
			name:          "ranking point id is not an identifier",
			modify:        func(y *GameYAML) { y.RPs[0].ID = "auto rp" },
			expectedError: "ranking_points[0]: id 'auto rp' must be a valid identifier",
		},
		{
			name:          "ranking point without a logic func",
			modify:        func(y *GameYAML) { y.RPs[0].LogicFunc = "" },
			expectedError: "ranking_points[0].logic_func",
		},
		{
			name:          "ranking point logic func is not an identifier",
			modify:        func(y *GameYAML) { y.RPs[0].LogicFunc = "func with spaces" },
			expectedError: "ranking_points[0].logic_func",
		},
		{
			name: "duplicate id across sections",
			modify: func(y *GameYAML) {
				// A scoring count reusing the status id "park".
				y.ScoringCounts = append(
					y.ScoringCounts,
					ScoringCount{ID: "park", GamePiece: "cube", Phases: []PhasePoints{{Phase: "auto", Points: 5}}},
				)
			},
			expectedError: "duplicate id: 'park'",
		},
		{
			name: "unknown ranking tiebreaker metric",
			modify: func(y *GameYAML) {
				y.RankingTiebreakers = append(y.RankingTiebreakers, Tiebreaker{Metric: "nonexistent"})
			},
			expectedError: "ranking_tiebreakers[2]: unknown metric 'nonexistent'",
		},
		{
			name: "unknown playoff tiebreaker metric",
			modify: func(y *GameYAML) {
				y.PlayoffTiebreakers = append(y.PlayoffTiebreakers, Tiebreaker{Metric: "nonexistent"})
			},
			expectedError: "playoff_tiebreakers[2]: unknown metric 'nonexistent'",
		},
		{
			name: "a grouped scoring count is not itself a metric",
			modify: func(y *GameYAML) {
				// rack_low rolls up into the "rack" bucket, so it has no bucket of its own.
				y.RankingTiebreakers = append(y.RankingTiebreakers, Tiebreaker{Metric: "rack_low"})
			},
			expectedError: "unknown metric 'rack_low'",
		},
		{
			name: "duplicate ranking tiebreaker metric",
			modify: func(y *GameYAML) {
				y.RankingTiebreakers = append(y.RankingTiebreakers, Tiebreaker{Metric: "total_points"})
			},
			expectedError: "ranking_tiebreakers[2]: duplicate metric 'total_points'",
		},
		{
			name: "duplicate playoff tiebreaker metric",
			modify: func(y *GameYAML) {
				y.PlayoffTiebreakers = append(y.PlayoffTiebreakers, Tiebreaker{Metric: "auto_points"})
			},
			expectedError: "playoff_tiebreakers[2]: duplicate metric 'auto_points'",
		},
	}

	// One case per reserved word, for each of the three sections that contribute metric names.
	for _, reserved := range []string{
		"auto_points", "teleop_points", "endgame_points", "total_points",
		"score", "match_points", "foul_points", "ranking_points",
	} {
		reserved := reserved
		tests = append(
			tests,
			validationCase{
				name:          "reserved scoring group id " + reserved,
				modify:        func(y *GameYAML) { y.ScoringGroups[0].ID = reserved },
				expectedError: "scoring_groups[0]: id '" + reserved + "' is reserved",
			},
			validationCase{
				name:          "reserved scoring count id " + reserved,
				modify:        func(y *GameYAML) { y.ScoringCounts[2].ID = reserved },
				expectedError: "scoring_counts[2]: id '" + reserved + "' is reserved",
			},
			validationCase{
				name:          "reserved status id " + reserved,
				modify:        func(y *GameYAML) { y.Statuses[0].ID = reserved },
				expectedError: "statuses[0]: id '" + reserved + "' is reserved",
			},
		)
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Start from a fresh copy of the self-contained fixture, so appends in one case
				// don't leak into another.
				cfg := testGameYAML()
				tt.modify(cfg)

				validationErrors := ValidateGameYAML(cfg)
				if !assert.NotEmpty(t, validationErrors) {
					return
				}
				found := false
				for _, errStr := range validationErrors {
					if strings.Contains(errStr, tt.expectedError) {
						found = true
						break
					}
				}
				assert.True(
					t, found, "expected an error containing %q, got: %v", tt.expectedError, validationErrors,
				)
			},
		)
	}
}

func TestMetricIDs(t *testing.T) {
	cfg := testGameYAML()

	// Built-ins first, then group buckets in config order, then ungrouped counts, then statuses.
	assert.Equal(
		t,
		[]string{
			"auto_points", "teleop_points", "endgame_points", "total_points",
			"rack", "solo", "park", "climb",
		},
		cfg.MetricIDs(),
	)

	// A nil config still reports the built-ins, so validation of a config-free build is sane.
	var nilCfg *GameYAML
	assert.Equal(
		t, []string{"auto_points", "teleop_points", "endgame_points", "total_points"}, nilCfg.MetricIDs(),
	)
}

func TestMetricLabel(t *testing.T) {
	cfg := testGameYAML()

	assert.Equal(t, "Auto Points", cfg.MetricLabel("auto_points"))
	assert.Equal(t, "Teleop Points", cfg.MetricLabel("teleop_points"))
	assert.Equal(t, "Endgame Points", cfg.MetricLabel("endgame_points"))
	assert.Equal(t, "Total Points", cfg.MetricLabel("total_points"))
	assert.Equal(t, "Rack", cfg.MetricLabel("rack"))
	assert.Equal(t, "Solo", cfg.MetricLabel("solo"))
	assert.Equal(t, "Park", cfg.MetricLabel("park"))
	assert.Equal(t, "Climb", cfg.MetricLabel("climb"))

	// Unknown ids and configs fall back to the raw id rather than an empty column header.
	assert.Equal(t, "mystery", cfg.MetricLabel("mystery"))
	var nilCfg *GameYAML
	assert.Equal(t, "rack", nilCfg.MetricLabel("rack"))
	assert.Equal(t, "Auto Points", nilCfg.MetricLabel("auto_points"))
}

func TestMetricIDsCoversEveryValidTiebreaker(t *testing.T) {
	// Every metric MetricIDs reports must be accepted as a tiebreaker, and nothing else may be.
	cfg := testGameYAML()
	for _, id := range cfg.MetricIDs() {
		c := testGameYAML()
		c.RankingTiebreakers = []Tiebreaker{{Metric: id}}
		c.PlayoffTiebreakers = []Tiebreaker{{Metric: id}}
		assert.Empty(t, ValidateGameYAML(c), "metric %q should be a valid tiebreaker", id)
	}
}

func TestReadGameConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run(
		"missing file", func(t *testing.T) {
			_, err := ReadGameConfig(filepath.Join(dir, "nope.yaml"))
			assert.ErrorContains(t, err, "error reading")
		},
	)

	t.Run(
		"malformed yaml", func(t *testing.T) {
			path := filepath.Join(dir, "bad.yaml")
			assert.Nil(t, os.WriteFile(path, []byte("game: [oops\n"), 0644))
			_, err := ReadGameConfig(path)
			assert.ErrorContains(t, err, "error parsing")
		},
	)

	t.Run(
		"invalid config", func(t *testing.T) {
			path := filepath.Join(dir, "invalid.yaml")
			assert.Nil(t, os.WriteFile(path, []byte("game:\n  name: \"\"\n"), 0644))
			_, err := ReadGameConfig(path)
			assert.ErrorContains(t, err, "validation errors")
			assert.ErrorContains(t, err, "game.name is required")
		},
	)

	t.Run(
		"does not activate", func(t *testing.T) {
			before := GetActiveConfig()
			path := filepath.Join(dir, "good.yaml")
			assert.Nil(
				t,
				os.WriteFile(
					path,
					[]byte("game:\n  name: Read Only\nfouls:\n  minor_foul_points: 1\n  major_foul_points: 2\n"),
					0644,
				),
			)
			cfg, err := ReadGameConfig(path)
			assert.Nil(t, err)
			assert.Equal(t, "Read Only", cfg.Game.Name)
			assert.Equal(t, before, GetActiveConfig())
		},
	)
}

func TestConfigLookupIndexes(t *testing.T) {
	cfg := testGameYAML()
	previous := GetActiveConfig()
	SetActiveConfig(cfg)
	t.Cleanup(func() { SetActiveConfig(previous) })

	assert.Equal(t, "rack_low", cfg.Count("rack_low").ID)
	assert.Nil(t, cfg.Count("nonexistent"))
	assert.Equal(t, "climb", cfg.Status("climb").ID)
	assert.Nil(t, cfg.Status("nonexistent"))

	// Buckets: a grouped count reports its group, an ungrouped one reports itself.
	assert.Equal(t, "rack", cfg.Count("rack_low").Bucket())
	assert.Equal(t, "solo", cfg.Count("solo").Bucket())

	// A config that was never activated builds its indexes lazily.
	fresh := testGameYAML()
	assert.Equal(t, "rack_high", fresh.Count("rack_high").ID)
	assert.Equal(t, "park", fresh.Status("park").ID)

	var nilCfg *GameYAML
	assert.Nil(t, nilCfg.Count("rack_low"))
	assert.Nil(t, nilCfg.Status("park"))
}
