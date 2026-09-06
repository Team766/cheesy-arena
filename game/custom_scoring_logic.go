//go:build custom

package game

// Bonus ranking-point logic for the game defined in custom_game.yaml. Each function is registered
// under the logic_func name the YAML uses; the server refuses to start if one is missing.
func init() {
	RegisterLogicFunc("ComputeAdventurerRP", ComputeAdventurerRP)
	RegisterLogicFunc("ComputeExplorerRP", ComputeExplorerRP)
	RegisterLogicFunc("ComputeSummitRP", ComputeSummitRP)
}

// AdventurerPointThreshold is the combined shelf and chest score that earns the Adventurer bonus.
const AdventurerPointThreshold = 30

// ComputeAdventurerRP: the alliance scored at least AdventurerPointThreshold points worth of gems
// across the shelf (either layer) and the treasure chest, in any phase.
func ComputeAdventurerRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return summary.GroupPoints["shelf"]+summary.GroupPoints["chest"] >= AdventurerPointThreshold
}

// ComputeExplorerRP: all three robots crossed the bridge during auto.
func ComputeExplorerRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.CountBoolStatus("bridge") == 3
}

// ComputeSummitRP: at least two robots finished on the ledge or the summit (ascent value index 1
// is "ledge", 2 is "summit").
func ComputeSummitRP(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.CountEnumStatus("ascent", 1) >= 2
}
