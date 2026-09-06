//go:build custom

package web

import (
	"encoding/json"
	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

// The scoring, referee and audience panels build their DOM from this endpoint, so it must serve the
// active config verbatim, with the YAML-style keys.
func TestGameConfigApi(t *testing.T) {
	web := setupTestWeb(t)
	cfg := game.GetActiveConfig()
	if !assert.NotNil(t, cfg) {
		return
	}

	recorder := web.getHttpResponse("/api/game_config")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header()["Content-Type"][0])

	var served game.GameYAML
	assert.Nil(t, json.Unmarshal(recorder.Body.Bytes(), &served))
	assert.Equal(t, cfg.Game.Name, served.Game.Name)
	assert.Equal(t, cfg.Fouls, served.Fouls)
	assert.Equal(t, cfg.GamePieces, served.GamePieces)
	assert.Equal(t, cfg.ScoringGroups, served.ScoringGroups)
	assert.Equal(t, cfg.ScoringCounts, served.ScoringCounts)
	assert.Equal(t, cfg.Statuses, served.Statuses)
	assert.Equal(t, cfg.RPs, served.RPs)
	assert.Equal(t, cfg.RankingTiebreakers, served.RankingTiebreakers)
	assert.Equal(t, cfg.PlayoffTiebreakers, served.PlayoffTiebreakers)
}

func TestGameConfigApiWithoutAConfig(t *testing.T) {
	web := setupTestWeb(t)

	previous := game.GetActiveConfig()
	game.SetActiveConfig(nil)
	t.Cleanup(func() { game.SetActiveConfig(previous) })

	recorder := web.getHttpResponse("/api/game_config")
	assert.Equal(t, 404, recorder.Code)
	assert.Equal(t, "{}", recorder.Body.String())
}
