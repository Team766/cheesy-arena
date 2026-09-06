//go:build custom

package field

import (
	"github.com/Team254/cheesy-arena/game"
	"testing"
)

// setupTestGameConfig makes the checked-in fixture config active for the duration of the test, so
// that every package building an arena in a test (field, web, model, tournament) scores against a
// known game rather than whichever game/custom_game.yaml happens to be checked in.
func setupTestGameConfig(t *testing.T) {
	game.LoadFixtureConfig(t)
}
