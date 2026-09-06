//go:build !custom

package web

import "github.com/Team254/cheesy-arena/game"

// buildMatchReviewEditPhases has nothing to build in the standard FRC build: edit_match_result.html
// hardcodes that game's score-editing form rather than deriving it from a config.
func buildMatchReviewEditPhases(*game.Score) []MatchReviewEditPhase {
	return nil
}
