//go:build custom

package game

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestJsonContract pins the exact top-level JSON key set of the three structs the web UI decodes.
// The JavaScript in static/js reads these names directly, and nothing else in the Go test suite
// would catch a rename or a stray json tag, so treat a failure here as a UI-breaking change: fix
// the struct, or update the JS and this test together.
func TestJsonContract(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	t.Run(
		"Score", func(t *testing.T) {
			assert.ElementsMatch(
				t,
				[]string{"Counts", "BoolStatuses", "EnumStatuses", "Fouls", "PlayoffDq", "Hub"},
				topLevelKeys(t, scoreEverything(t, cfg)),
			)
		},
	)

	t.Run(
		"ScoreSummary", func(t *testing.T) {
			assert.ElementsMatch(
				t,
				[]string{
					"PlayoffDq", "AutoPoints", "TeleopPoints", "EndgamePoints", "MatchPoints",
					"FoulPoints", "Score", "NumOpponentMajorFouls", "GroupPoints", "StatusPoints",
					"RPs", "BonusRankingPoints",
				},
				topLevelKeys(t, scoreEverything(t, cfg).Summarize(nil)),
			)
		},
	)

	t.Run(
		"Ranking", func(t *testing.T) {
			assert.ElementsMatch(
				t,
				[]string{
					"TeamId", "Rank", "PreviousRank", "RankingPoints", "Tiebreakers", "Random",
					"Wins", "Losses", "Ties", "Disqualifications", "Played",
				},
				topLevelKeys(t, TestRanking1()),
			)
		},
	)

	// The nested maps are keyed by config ids, not by index, so the UI can look a bucket up by name.
	summary := scoreEverything(t, cfg).Summarize(nil)
	for _, sc := range cfg.ScoringCounts {
		assert.Contains(t, summary.GroupPoints, sc.Bucket())
	}
	for _, st := range cfg.Statuses {
		assert.Contains(t, summary.StatusPoints, st.ID)
	}
	for _, rp := range cfg.RPs {
		assert.Contains(t, summary.RPs, rp.ID)
	}

	// Score.Counts is keyed "<countId>_<phase>".
	score := new(Score)
	assert.True(t, score.AdjustCount(cfg.ScoringCounts[0].ID, PhaseAuto, 1))
	assert.Contains(t, score.Counts, cfg.ScoringCounts[0].ID+"_auto")
}

// TestGameConfigJsonKeysAreSnakeCase pins the other half of the contract: the config document served
// by GET /api/game_config keeps its lowercase YAML-style keys, because the panels build their DOM
// from the same field names the designer typed into custom_game.yaml.
func TestGameConfigJsonKeysAreSnakeCase(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	assert.ElementsMatch(
		t,
		[]string{
			"game", "fouls", "game_pieces", "scoring_groups", "scoring_counts", "statuses",
			"ranking_points", "ranking_tiebreakers", "playoff_tiebreakers",
		},
		topLevelKeys(t, cfg),
	)

	data, err := json.Marshal(cfg)
	assert.Nil(t, err)
	var decoded struct {
		ScoringCounts []struct {
			ID           string `json:"id"`
			DisplayName  string `json:"display_name"`
			GamePiece    string `json:"game_piece"`
			ScoringGroup string `json:"scoring_group"`
			Phases       []struct {
				Phase  string `json:"phase"`
				Points int    `json:"points"`
			} `json:"phases"`
		} `json:"scoring_counts"`
		Statuses []struct {
			ID     string `json:"id"`
			Values []struct {
				ID     string `json:"id"`
				Points int    `json:"points"`
			} `json:"values"`
		} `json:"statuses"`
	}
	assert.Nil(t, json.Unmarshal(data, &decoded))
	assert.Len(t, decoded.ScoringCounts, len(cfg.ScoringCounts))
	assert.Equal(t, cfg.ScoringCounts[0].ID, decoded.ScoringCounts[0].ID)
	assert.Equal(t, cfg.ScoringCounts[0].DisplayName, decoded.ScoringCounts[0].DisplayName)
	assert.Equal(t, cfg.ScoringCounts[0].GamePiece, decoded.ScoringCounts[0].GamePiece)
	assert.Equal(t, cfg.ScoringCounts[0].Phases[0].Points, decoded.ScoringCounts[0].Phases[0].Points)
	assert.Len(t, decoded.Statuses, len(cfg.Statuses))
}

func topLevelKeys(t *testing.T, value any) []string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal %T: %v", value, err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode %T as a JSON object: %v", value, err)
	}
	keys := make([]string, 0, len(decoded))
	for key := range decoded {
		keys = append(keys, key)
	}
	return keys
}
