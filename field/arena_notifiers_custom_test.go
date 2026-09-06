//go:build custom

package field

import (
	"encoding/json"
	"testing"

	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
)

// The realtime score message is serialized by every websocket listener concurrently with scoring
// panel mutations, so it must carry a snapshot of the score rather than a pointer into the live one.
func TestRealtimeScoreMessageSnapshotsTheScore(t *testing.T) {
	arena := SetupTestArena(t)
	cfg := game.GetActiveConfig()
	count := cfg.ScoringCounts[0]
	phase, ok := game.PhaseFromString(count.Phases[0].Phase)
	assert.True(t, ok)

	message := arena.generateRealtimeScoreMessage()
	before, err := json.Marshal(message)
	assert.Nil(t, err)

	assert.True(t, arena.RedRealtimeScore.CurrentScore.AdjustCount(count.ID, phase, 3))
	after, err := json.Marshal(message)
	assert.Nil(t, err)
	assert.Equal(t, string(before), string(after))

	// A fresh message sees the mutation.
	fresh, err := json.Marshal(arena.generateRealtimeScoreMessage())
	assert.Nil(t, err)
	assert.NotEqual(t, string(before), string(fresh))
}
