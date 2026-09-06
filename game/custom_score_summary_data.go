//go:build custom

package game

type ScoreSummary struct {
	PlayoffDq             bool            `json:"playoff_dq"`
	AutoPoints            int             `json:"auto_points"`
	TeleopPoints          int             `json:"teleop_points"`
	EndgamePoints         int             `json:"endgame_points"`
	MatchPoints           int             `json:"match_points"`
	FoulPoints            int             `json:"foul_points"`
	Score                 int             `json:"score"`
	NumOpponentMajorFouls int             `json:"num_opponent_major_fouls"`
	GroupPoints           map[string]int  `json:"group_points"`
	StatusPoints          map[string]int  `json:"status_points"`
	RPs                   map[string]bool `json:"rps"`
	BonusRankingPoints    int             `json:"bonus_ranking_points"`
}

func (score *Score) Summarize(opponentScore *Score) *ScoreSummary {
	summary := new(ScoreSummary)
	summary.GroupPoints = make(map[string]int)
	summary.StatusPoints = make(map[string]int)
	summary.RPs = make(map[string]bool)

	if score == nil {
		return summary
	}

	summary.PlayoffDq = score.PlayoffDq
	if score.PlayoffDq {
		return summary
	}

	cfg := GetActiveConfig()
	if cfg != nil {
		for _, sc := range cfg.ScoringCounts {
			bucketID := sc.ScoringGroup
			if bucketID == "" {
				bucketID = sc.ID
			}
			for _, pp := range sc.Phases {
				var phaseEnum Phase
				switch pp.Phase {
				case "auto":
					phaseEnum = PhaseAuto
				case "teleop":
					phaseEnum = PhaseTeleop
				case "endgame":
					phaseEnum = PhaseEndgame
				}
				count := score.GetCount(sc.ID, phaseEnum)
				pts := count * pp.Points

				switch pp.Phase {
				case "auto":
					summary.AutoPoints += pts
				case "teleop":
					summary.TeleopPoints += pts
				case "endgame":
					summary.EndgamePoints += pts
				}
				summary.GroupPoints[bucketID] += pts
			}
		}

		for _, st := range cfg.Statuses {
			phaseStr := ""
			if len(st.Phases) > 0 {
				phaseStr = st.Phases[0].Phase
			}

			pts := 0
			if len(st.Values) > 0 {
				for robotIdx := 0; robotIdx < 3; robotIdx++ {
					valIdx := score.GetEnumStatus(st.ID, robotIdx)
					if valIdx >= 0 && valIdx < len(st.Values) {
						pts += st.Values[valIdx].Points
					}
				}
			} else if len(st.Phases) > 0 {
				numTrue := score.CountBoolStatus(st.ID)
				pts = numTrue * st.Phases[0].Points
			}

			summary.StatusPoints[st.ID] = pts
			switch phaseStr {
			case "auto":
				summary.AutoPoints += pts
			case "endgame":
				summary.EndgamePoints += pts
			}
		}
	}

	summary.MatchPoints = summary.AutoPoints + summary.TeleopPoints + summary.EndgamePoints

	if opponentScore != nil {
		for _, foul := range opponentScore.Fouls {
			summary.FoulPoints += foul.PointValue()
			if foul.IsMajor {
				summary.NumOpponentMajorFouls++
			}
		}
	}

	summary.Score = summary.MatchPoints + summary.FoulPoints

	if cfg != nil {
		for _, rp := range cfg.RPs {
			handler := Handlers[rp.LogicFunc]
			achieved := false
			if handler != nil {
				achieved = handler(score, opponentScore, summary)
			}
			summary.RPs[rp.ID] = achieved
			if achieved {
				summary.BonusRankingPoints++
			}
		}
	}

	return summary
}

func (summary *ScoreSummary) GetMetric(metric string) int {
	if summary == nil {
		return 0
	}
	switch metric {
	case "auto_points":
		return summary.AutoPoints
	case "teleop_points":
		return summary.TeleopPoints
	case "endgame_points":
		return summary.EndgamePoints
	case "total_points", "match_points":
		return summary.MatchPoints
	case "score":
		return summary.Score
	case "foul_points":
		return summary.FoulPoints
	}
	if pts, ok := summary.GroupPoints[metric]; ok {
		return pts
	}
	if pts, ok := summary.StatusPoints[metric]; ok {
		return pts
	}
	return 0
}

func DetermineMatchStatus(
	redSummary, blueSummary *ScoreSummary,
	applyPlayoffTiebreakers bool,
) (MatchStatus, string) {
	if redSummary.PlayoffDq != blueSummary.PlayoffDq {
		if redSummary.PlayoffDq {
			return BlueWonMatch, ""
		}
		return RedWonMatch, ""
	}

	if redSummary.Score > blueSummary.Score {
		return RedWonMatch, ""
	} else if blueSummary.Score > redSummary.Score {
		return BlueWonMatch, ""
	}

	if applyPlayoffTiebreakers {
		// Tiebreaker 1: Opponent major fouls
		if redSummary.NumOpponentMajorFouls > blueSummary.NumOpponentMajorFouls {
			return RedWonMatch, "TIEBREAK: MAJOR FOULS"
		} else if blueSummary.NumOpponentMajorFouls > redSummary.NumOpponentMajorFouls {
			return BlueWonMatch, "TIEBREAK: MAJOR FOULS"
		}

		cfg := GetActiveConfig()
		if cfg != nil {
			for _, tb := range cfg.PlayoffTiebreakers {
				redVal := redSummary.GetMetric(tb.Metric)
				blueVal := blueSummary.GetMetric(tb.Metric)
				if redVal > blueVal {
					return RedWonMatch, "TIEBREAK: " + tb.Metric
				} else if blueVal > redVal {
					return BlueWonMatch, "TIEBREAK: " + tb.Metric
				}
			}
		}
		return TieMatch, "TRUE TIE"
	}

	return TieMatch, ""
}
