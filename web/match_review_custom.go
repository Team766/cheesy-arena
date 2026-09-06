//go:build custom

// The game-specific half of the match review edit page for a custom game: the score-editing form is
// derived from the active YAML config and rendered server-side, so that every count and status
// comes back pre-populated with the stored value without the page having to fetch the config.

package web

import "github.com/Team254/cheesy-arena/game"

// matchReviewEditPhaseNames gives the phases their section headings, in the order they are shown.
var matchReviewEditPhaseNames = []struct {
	id          string
	displayName string
}{
	{"auto", "Auto"},
	{"teleop", "Teleop"},
	{"endgame", "Endgame"},
}

// buildMatchReviewEditPhases turns the active game config, plus one alliance's stored score, into
// the per-phase form model that edit_match_result.html renders. Phases that the config doesn't use
// are omitted entirely.
func buildMatchReviewEditPhases(score *game.Score) []MatchReviewEditPhase {
	cfg := game.GetActiveConfig()
	if cfg == nil || score == nil {
		return nil
	}

	var phases []MatchReviewEditPhase
	for _, phaseName := range matchReviewEditPhaseNames {
		phaseEnum, ok := game.PhaseFromString(phaseName.id)
		if !ok {
			continue
		}
		phase := MatchReviewEditPhase{Id: phaseName.id, DisplayName: phaseName.displayName}

		for _, count := range cfg.ScoringCounts {
			for _, phasePoints := range count.Phases {
				if phasePoints.Phase != phaseName.id {
					continue
				}
				phase.Counts = append(
					phase.Counts,
					MatchReviewEditCount{
						Key:         count.ID + "_" + phaseName.id,
						DisplayName: count.DisplayName,
						Value:       score.GetCount(count.ID, phaseEnum),
					},
				)
			}
		}

		for _, status := range cfg.Statuses {
			if len(status.Phases) == 0 || status.Phases[0].Phase != phaseName.id {
				continue
			}
			if len(status.Values) == 0 {
				boolStatus := MatchReviewEditBoolStatus{Id: status.ID, DisplayName: status.DisplayName}
				for i := range boolStatus.Values {
					boolStatus.Values[i] = score.GetBoolStatus(status.ID, i)
				}
				phase.BoolStatuses = append(phase.BoolStatuses, boolStatus)
			} else {
				enumStatus := MatchReviewEditEnumStatus{Id: status.ID, DisplayName: status.DisplayName}
				for i, value := range status.Values {
					enumStatus.Options = append(
						enumStatus.Options, MatchReviewEditEnumOption{Index: i, DisplayName: value.DisplayName},
					)
				}
				for i := range enumStatus.Values {
					enumStatus.Values[i] = score.GetEnumStatus(status.ID, i)
				}
				phase.EnumStatuses = append(phase.EnumStatuses, enumStatus)
			}
		}

		if len(phase.Counts) > 0 || len(phase.BoolStatuses) > 0 || len(phase.EnumStatuses) > 0 {
			phases = append(phases, phase)
		}
	}

	return phases
}
