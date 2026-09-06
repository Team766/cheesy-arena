//go:build custom

package game

func init() {
	RegisterLogicFunc("ComputeAutonRP", ComputeAutonRP)
	RegisterLogicFunc("ComputeScoringRP", ComputeScoringRP)
	RegisterLogicFunc("ComputeEndgameRP", ComputeEndgameRP)
}

// ComputeAutonRP: alliance places more than 2 game pieces on Structure 1 (either level) during auto.
func ComputeAutonRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.GetCount("structure1_level1", PhaseAuto)+score.GetCount("structure1_level2", PhaseAuto) > 2
}

// ComputeScoringRP: alliance places 10 or more game pieces on Structure 1 during teleop.
func ComputeScoringRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.GetCount("structure1_level1", PhaseTeleop)+score.GetCount("structure1_level2", PhaseTeleop) >= 10
}

// ComputeEndgameRP: alliance parks at least two of three robots.
func ComputeEndgameRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.CountBoolStatus("park") >= 2
}
