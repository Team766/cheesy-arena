//go:build custom

package game

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUniqueMatchSounds(t *testing.T) {
	UpdateMatchSounds()

	uniqueSounds := UniqueMatchSounds()

	assert.Equal(
		t,
		[]string{
			"start",
			"end",
			"resume",
			"warning",
			"abort",
			"match_result",
			"pick_clock",
			"pick_clock_expired",
			"field_reset",
		},
		matchSoundNames(uniqueSounds),
	)
	assert.Len(t, uniqueSounds, 9)
}

func matchSoundNames(matchSounds []*MatchSound) []string {
	names := make([]string, 0, len(matchSounds))
	for _, sound := range matchSounds {
		names = append(names, sound.Name)
	}
	return names
}
