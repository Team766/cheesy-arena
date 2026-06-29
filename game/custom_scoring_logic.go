//go:build custom

package game

// Custom scoring logic for the active custom game. Hand-written; never generated or touched by
// `go generate`. Every logic_func named in custom_game.yaml's ranking_points needs a matching func
// here with the signature `func(score Score, opponentScore Score) bool` — the examples below show
// the pattern (they reference fields from the generated Score struct).

// ComputeAutonRP: alliance places more than 2 game pieces on Structure 1 (either level) during auto.
func ComputeAutonRP(score Score, opponentScore Score) bool {
	return score.AutoStructure1Level1Count+score.AutoStructure1Level2Count > 2
}

// ComputeScoringRP: alliance places 10 or more game pieces on Structure 1 during teleop.
func ComputeScoringRP(score Score, opponentScore Score) bool {
	return score.TeleopStructure1Level1Count+score.TeleopStructure1Level2Count >= 10
}

// ComputeEndgameRP: alliance parks at least two of three robots.
func ComputeEndgameRP(score Score, opponentScore Score) bool {
	parked := 0
	for _, p := range score.ParkStatuses {
		if p {
			parked++
		}
	}
	return parked >= 2
}
