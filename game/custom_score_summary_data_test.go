//go:build custom

package game

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// phasePointsField returns the summary phase total that the named YAML phase feeds.
func phasePointsField(summary *ScoreSummary, phase string) int {
	switch phase {
	case "auto":
		return summary.AutoPoints
	case "teleop":
		return summary.TeleopPoints
	case "endgame":
		return summary.EndgamePoints
	}
	return 0
}

// TestSummarizeOneCountAtATime is the core config-derived invariant: for every scoring count and
// every phase it declares, scoring exactly one of it must move the phase total, its group bucket and
// the match total by exactly the points the config assigns — and nothing else.
func TestSummarizeOneCountAtATime(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	for _, sc := range cfg.ScoringCounts {
		for _, pp := range sc.Phases {
			sc, pp := sc, pp
			t.Run(
				sc.ID+"/"+pp.Phase, func(t *testing.T) {
					phase, _ := PhaseFromString(pp.Phase)
					score := new(Score)
					assert.True(t, score.AdjustCount(sc.ID, phase, 1))

					summary := score.Summarize(nil)

					assert.Equal(t, pp.Points, phasePointsField(summary, pp.Phase))
					assert.Equal(t, pp.Points, summary.GroupPoints[sc.Bucket()])
					assert.Equal(t, pp.Points, summary.MatchPoints)
					assert.Equal(t, pp.Points, summary.Score)

					// The other two phase totals stay at zero.
					for _, other := range []string{"auto", "teleop", "endgame"} {
						if other != pp.Phase {
							assert.Equal(t, 0, phasePointsField(summary, other), "phase %s", other)
						}
					}
					// Every other bucket stays at zero.
					for _, other := range cfg.ScoringCounts {
						if other.Bucket() != sc.Bucket() {
							assert.Equal(t, 0, summary.GroupPoints[other.Bucket()], other.Bucket())
						}
					}
					// Scaling the count scales the points linearly.
					assert.True(t, score.AdjustCount(sc.ID, phase, 4))
					assert.Equal(t, 5*pp.Points, score.Summarize(nil).MatchPoints)
				},
			)
		}
	}
}

// TestSummarizeAllCountsAtOnce is the mutation-proof for the bucket rollup: with one of everything
// scored, each bucket must equal the sum of its members' points and the phase totals must partition
// the match total.
func TestSummarizeAllCountsAtOnce(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	score := new(Score)
	expectedByBucket := make(map[string]int)
	expectedByPhase := make(map[string]int)
	for _, sc := range cfg.ScoringCounts {
		for _, pp := range sc.Phases {
			phase, _ := PhaseFromString(pp.Phase)
			assert.True(t, score.AdjustCount(sc.ID, phase, 2))
			expectedByBucket[sc.Bucket()] += 2 * pp.Points
			expectedByPhase[pp.Phase] += 2 * pp.Points
		}
	}

	summary := score.Summarize(nil)
	for bucket, points := range expectedByBucket {
		assert.Equal(t, points, summary.GroupPoints[bucket], bucket)
	}
	assert.Equal(t, expectedByPhase["auto"], summary.AutoPoints)
	assert.Equal(t, expectedByPhase["teleop"], summary.TeleopPoints)
	assert.Equal(t, expectedByPhase["endgame"], summary.EndgamePoints)
	assert.Equal(
		t,
		expectedByPhase["auto"]+expectedByPhase["teleop"]+expectedByPhase["endgame"],
		summary.MatchPoints,
	)
	assert.Equal(t, summary.AutoPoints+summary.TeleopPoints+summary.EndgamePoints, summary.MatchPoints)
}

// TestSummarizeStatusPoints checks each status in isolation: bool statuses score phases[0].points
// per robot, enum statuses score the selected value's points per robot, and both land in the phase
// total the config names.
func TestSummarizeStatusPoints(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	for _, st := range cfg.Statuses {
		st := st
		t.Run(
			st.ID, func(t *testing.T) {
				phaseName := st.Phases[0].Phase

				// Nothing set: no points anywhere.
				assert.Equal(t, 0, new(Score).Summarize(nil).StatusPoints[st.ID])

				if len(st.Values) > 0 {
					for i, value := range st.Values {
						score := new(Score)
						assert.True(t, i == 0 || score.SetEnumStatus(st.ID, 0, i))
						summary := score.Summarize(nil)
						assert.Equal(t, value.Points, summary.StatusPoints[st.ID], value.ID)
						assert.Equal(t, value.Points, phasePointsField(summary, phaseName), value.ID)
						assert.Equal(t, value.Points, summary.MatchPoints, value.ID)
					}

					// All three robots at the top value.
					top := len(st.Values) - 1
					score := new(Score)
					for robot := 0; robot < 3; robot++ {
						assert.True(t, top == 0 || score.SetEnumStatus(st.ID, robot, top))
					}
					assert.Equal(t, 3*st.Values[top].Points, score.Summarize(nil).StatusPoints[st.ID])
				} else {
					for robots := 1; robots <= 3; robots++ {
						score := new(Score)
						for robot := 0; robot < robots; robot++ {
							assert.True(t, score.SetBoolStatus(st.ID, robot, true))
						}
						summary := score.Summarize(nil)
						assert.Equal(t, robots*st.Phases[0].Points, summary.StatusPoints[st.ID])
						assert.Equal(t, robots*st.Phases[0].Points, phasePointsField(summary, phaseName))
						assert.Equal(t, robots*st.Phases[0].Points, summary.MatchPoints)
					}
				}

				// Statuses never contribute to a bucket.
				for _, sc := range cfg.ScoringCounts {
					score := new(Score)
					if len(st.Values) > 0 {
						score.SetEnumStatus(st.ID, 0, len(st.Values)-1)
					} else {
						score.SetBoolStatus(st.ID, 0, true)
					}
					assert.Equal(t, 0, score.Summarize(nil).GroupPoints[sc.Bucket()])
				}
			},
		)
	}
}

// TestSummarizeFoulPoints checks that fouls score for the OPPONENT, at the config's point values.
func TestSummarizeFoulPoints(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	assert.Equal(t, cfg.Fouls.MinorFoulPoints, MinorFoulPoints)
	assert.Equal(t, cfg.Fouls.MajorFoulPoints, MajorFoulPoints)

	score := new(Score)
	opponent := &Score{
		Fouls: []Foul{
			{FoulId: 1, IsMajor: false, TeamId: 254},
			{FoulId: 2, IsMajor: true, TeamId: 254},
			{FoulId: 3, IsMajor: true, TeamId: 1114},
		},
	}

	summary := score.Summarize(opponent)
	expected := cfg.Fouls.MinorFoulPoints + 2*cfg.Fouls.MajorFoulPoints
	assert.Equal(t, expected, summary.FoulPoints)
	assert.Equal(t, 2, summary.NumOpponentMajorFouls)
	assert.Equal(t, 0, summary.MatchPoints)
	assert.Equal(t, expected, summary.Score)

	// The alliance's own fouls do not score for itself.
	assert.Equal(t, 0, opponent.Summarize(nil).FoulPoints)
	assert.Equal(t, 0, new(Score).Summarize(nil).FoulPoints)
}

func TestSummarizePlayoffDq(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	score := scoreEverything(t, cfg)
	score.PlayoffDq = true

	summary := score.Summarize(&Score{Fouls: []Foul{{IsMajor: true}}})
	assert.True(t, summary.PlayoffDq)
	assert.Equal(t, 0, summary.Score)
	assert.Equal(t, 0, summary.MatchPoints)
	assert.Equal(t, 0, summary.FoulPoints)
	assert.Empty(t, summary.RPs)
	assert.Equal(t, 0, summary.BonusRankingPoints)

	// A nil score summarizes to an empty, non-nil summary.
	var nilScore *Score
	empty := nilScore.Summarize(nil)
	assert.NotNil(t, empty.GroupPoints)
	assert.NotNil(t, empty.StatusPoints)
	assert.NotNil(t, empty.RPs)
}

// TestRankingPointDispatch checks that every ranking point in the config is dispatched to its
// registered logic function and counted into BonusRankingPoints.
func TestRankingPointDispatch(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	assert.Empty(t, ValidateHandlers(cfg))

	// Every configured RP appears in the map, false by default.
	summary := new(Score).Summarize(nil)
	assert.Len(t, summary.RPs, len(cfg.RPs))
	for _, rp := range cfg.RPs {
		achieved, present := summary.RPs[rp.ID]
		assert.True(t, present, rp.ID)
		assert.False(t, achieved, rp.ID)
	}
	assert.Equal(t, 0, summary.BonusRankingPoints)

	// Drive each RP true one at a time by swapping in a stub handler, and check that
	// BonusRankingPoints counts exactly the achieved ones.
	for _, rp := range cfg.RPs {
		rp := rp
		t.Run(
			rp.ID, func(t *testing.T) {
				original := Handlers[rp.LogicFunc]
				Handlers[rp.LogicFunc] = func(score, opponentScore *Score, summary *ScoreSummary) bool {
					return true
				}
				t.Cleanup(func() { Handlers[rp.LogicFunc] = original })

				summary := new(Score).Summarize(nil)
				assert.True(t, summary.RPs[rp.ID])
				assert.Equal(t, 1, summary.BonusRankingPoints)
			},
		)
	}

	// All handlers true at once.
	for _, rp := range cfg.RPs {
		original := Handlers[rp.LogicFunc]
		Handlers[rp.LogicFunc] = func(score, opponentScore *Score, summary *ScoreSummary) bool { return true }
		defer func(name string, fn LogicFunc) { Handlers[name] = fn }(rp.LogicFunc, original)
	}
	assert.Equal(t, len(cfg.RPs), new(Score).Summarize(nil).BonusRankingPoints)
}

// TestRankingPointHandlersSeeTheRealScore exercises the fixture's own logic functions rather than
// stubs, confirming that the score, the opponent score and the partially-built summary are all
// threaded through.
func TestRankingPointHandlersSeeTheRealScore(t *testing.T) {
	LoadFixtureConfig(t)

	// FixtureAutoRp: summary.AutoPoints >= 6. rack_low is worth 3 in auto.
	score := new(Score)
	assert.True(t, score.AdjustCount("rack_low", PhaseAuto, 1))
	assert.False(t, score.Summarize(nil).RPs["auto_rp"])
	assert.True(t, score.AdjustCount("rack_low", PhaseAuto, 1))
	assert.True(t, score.Summarize(nil).RPs["auto_rp"])

	// FixtureClimbRp: any robot above the baseline climb value.
	climb := new(Score)
	assert.False(t, climb.Summarize(nil).RPs["climb_rp"])
	assert.True(t, climb.SetEnumStatusByID("climb", 2, "low"))
	assert.True(t, climb.Summarize(nil).RPs["climb_rp"])
}

func TestGetMetricCoversEveryMetricID(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	score := scoreEverything(t, cfg)
	summary := score.Summarize(&Score{Fouls: []Foul{{IsMajor: true}}})

	for _, id := range cfg.MetricIDs() {
		var expected int
		switch id {
		case "auto_points":
			expected = summary.AutoPoints
		case "teleop_points":
			expected = summary.TeleopPoints
		case "endgame_points":
			expected = summary.EndgamePoints
		case "total_points":
			expected = summary.MatchPoints
		default:
			if points, ok := summary.GroupPoints[id]; ok {
				expected = points
			} else {
				expected = summary.StatusPoints[id]
			}
		}
		assert.Equal(t, expected, summary.GetMetric(id), id)
		assert.NotZero(t, expected, "metric %q should be non-zero for a fully-scored fixture score", id)
	}

	// Names that validation reserves are NOT metrics: GetMetric and the validator agree on one
	// vocabulary.
	for _, reserved := range []string{"score", "match_points", "foul_points", "ranking_points", "bogus"} {
		assert.Equal(t, 0, summary.GetMetric(reserved), reserved)
	}

	var nilSummary *ScoreSummary
	assert.Equal(t, 0, nilSummary.GetMetric("auto_points"))
}

func TestDetermineMatchStatusWithoutTiebreakers(t *testing.T) {
	LoadFixtureConfig(t)

	red := &ScoreSummary{Score: 50}
	blue := &ScoreSummary{Score: 40}

	status, reason := DetermineMatchStatus(red, blue, false)
	assert.Equal(t, RedWonMatch, status)
	assert.Equal(t, "", reason)

	status, reason = DetermineMatchStatus(blue, red, false)
	assert.Equal(t, BlueWonMatch, status)
	assert.Equal(t, "", reason)

	status, reason = DetermineMatchStatus(red, &ScoreSummary{Score: 50}, false)
	assert.Equal(t, TieMatch, status)
	assert.Equal(t, "", reason)

	// A playoff DQ loses regardless of score.
	status, _ = DetermineMatchStatus(&ScoreSummary{Score: 99, PlayoffDq: true}, &ScoreSummary{}, true)
	assert.Equal(t, BlueWonMatch, status)
	status, _ = DetermineMatchStatus(&ScoreSummary{}, &ScoreSummary{Score: 99, PlayoffDq: true}, true)
	assert.Equal(t, RedWonMatch, status)
}

// TestPlayoffTiebreakerCascade walks the configured playoff tiebreakers in order: for each one, both
// alliances are equal on every earlier criterion and differ only on this one.
func TestPlayoffTiebreakerCascade(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	assert.NotEmpty(t, cfg.PlayoffTiebreakers)

	// Opponent major fouls always come first, ahead of anything the config lists.
	status, reason := DetermineMatchStatus(
		&ScoreSummary{Score: 50, NumOpponentMajorFouls: 1},
		&ScoreSummary{Score: 50, NumOpponentMajorFouls: 0},
		true,
	)
	assert.Equal(t, RedWonMatch, status)
	assert.Equal(t, "TIEBREAK: MAJOR FOULS", reason)

	status, reason = DetermineMatchStatus(
		&ScoreSummary{Score: 50, NumOpponentMajorFouls: 0},
		&ScoreSummary{Score: 50, NumOpponentMajorFouls: 2},
		true,
	)
	assert.Equal(t, BlueWonMatch, status)
	assert.Equal(t, "TIEBREAK: MAJOR FOULS", reason)

	for i, tb := range cfg.PlayoffTiebreakers {
		i, tb := i, tb
		t.Run(
			tb.Metric, func(t *testing.T) {
				// Equal on every earlier criterion, red ahead on this one.
				red := summaryWithMetrics(cfg, cfg.PlayoffTiebreakers[:i], 5)
				blue := summaryWithMetrics(cfg, cfg.PlayoffTiebreakers[:i], 5)
				setMetric(red, tb.Metric, 9)
				setMetric(blue, tb.Metric, 4)

				status, reason := DetermineMatchStatus(red, blue, true)
				assert.Equal(t, RedWonMatch, status)
				assert.Equal(t, "TIEBREAK: "+cfg.MetricLabel(tb.Metric), reason)

				status, reason = DetermineMatchStatus(blue, red, true)
				assert.Equal(t, BlueWonMatch, status)
				assert.Equal(t, "TIEBREAK: "+cfg.MetricLabel(tb.Metric), reason)
			},
		)
	}

	// Equal on everything: a true tie.
	red := summaryWithMetrics(cfg, cfg.PlayoffTiebreakers, 5)
	blue := summaryWithMetrics(cfg, cfg.PlayoffTiebreakers, 5)
	status, reason = DetermineMatchStatus(red, blue, true)
	assert.Equal(t, TieMatch, status)
	assert.Equal(t, "TRUE TIE", reason)
}

// summaryWithMetrics builds a summary whose named metrics all hold the given value.
func summaryWithMetrics(cfg *GameYAML, tiebreakers []Tiebreaker, value int) *ScoreSummary {
	summary := &ScoreSummary{
		Score:        50,
		GroupPoints:  make(map[string]int),
		StatusPoints: make(map[string]int),
		RPs:          make(map[string]bool),
	}
	for _, tb := range tiebreakers {
		setMetric(summary, tb.Metric, value)
	}
	return summary
}

// setMetric writes a value into whichever summary field GetMetric reads for the given metric id.
func setMetric(summary *ScoreSummary, metric string, value int) {
	switch metric {
	case "auto_points":
		summary.AutoPoints = value
	case "teleop_points":
		summary.TeleopPoints = value
	case "endgame_points":
		summary.EndgamePoints = value
	case "total_points":
		summary.MatchPoints = value
	default:
		cfg := GetActiveConfig()
		if cfg != nil && cfg.Status(metric) != nil {
			summary.StatusPoints[metric] = value
		} else {
			summary.GroupPoints[metric] = value
		}
	}
}

// TestRankingTiebreakerCascade drives the Rankings sort through every configured ranking
// tiebreaker, plus the RankingPoints-per-match primary and the random last resort.
func TestRankingTiebreakerCascade(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	assert.NotEmpty(t, cfg.RankingTiebreakers)

	newRanking := func(teamId int, rankingPoints int, values map[string]int, random float64) Ranking {
		tiebreakers := make(map[string]int)
		for k, v := range values {
			tiebreakers[k] = v
		}
		return Ranking{
			TeamId: teamId,
			RankingFields: RankingFields{
				RankingPoints: rankingPoints,
				Tiebreakers:   tiebreakers,
				Played:        10,
				Random:        random,
			},
		}
	}

	// Primary: ranking points per match played.
	rankings := Rankings{newRanking(2, 10, nil, 0.1), newRanking(1, 20, nil, 0.9)}
	assert.True(t, rankings.Less(1, 0))
	assert.False(t, rankings.Less(0, 1))

	// Each configured tiebreaker in turn, with everything before it equal.
	for i, tb := range cfg.RankingTiebreakers {
		equal := make(map[string]int)
		for _, earlier := range cfg.RankingTiebreakers[:i] {
			equal[earlier.Metric] = 7
		}
		ahead := newRanking(1, 20, equal, 0.1)
		ahead.Tiebreakers[tb.Metric] = 100
		behind := newRanking(2, 20, equal, 0.9)
		behind.Tiebreakers[tb.Metric] = 1

		pair := Rankings{ahead, behind}
		assert.True(t, pair.Less(0, 1), tb.Metric)
		assert.False(t, pair.Less(1, 0), tb.Metric)
	}

	// Everything equal: the random value breaks the tie, highest first.
	all := make(map[string]int)
	for _, tb := range cfg.RankingTiebreakers {
		all[tb.Metric] = 7
	}
	pair := Rankings{newRanking(1, 20, all, 0.9), newRanking(2, 20, all, 0.1)}
	assert.True(t, pair.Less(0, 1))
	assert.False(t, pair.Less(1, 0))

	assert.Equal(t, 2, pair.Len())
	pair.Swap(0, 1)
	assert.Equal(t, 2, pair[0].TeamId)
}

// TestAddScoreSummaryAccumulatesConfiguredTiebreakers checks that RankingFields accumulates exactly
// the metrics the config names as ranking tiebreakers.
func TestAddScoreSummaryAccumulatesConfiguredTiebreakers(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	score := scoreEverything(t, cfg)
	own := score.Summarize(nil)
	opponent := &ScoreSummary{Score: own.Score - 1}

	fields := &RankingFields{}
	fields.AddScoreSummary(own, opponent, false)

	assert.Equal(t, 1, fields.Played)
	assert.Equal(t, 1, fields.Wins)
	assert.Equal(t, 3+own.BonusRankingPoints, fields.RankingPoints)
	assert.Len(t, fields.Tiebreakers, len(cfg.RankingTiebreakers))
	for _, tb := range cfg.RankingTiebreakers {
		assert.Equal(t, own.GetMetric(tb.Metric), fields.Tiebreakers[tb.Metric], tb.Metric)
	}

	// A second match accumulates.
	fields.AddScoreSummary(own, opponent, false)
	for _, tb := range cfg.RankingTiebreakers {
		assert.Equal(t, 2*own.GetMetric(tb.Metric), fields.Tiebreakers[tb.Metric], tb.Metric)
	}

	// A tie awards one ranking point; a loss awards none; a DQ awards nothing at all.
	tied := &RankingFields{}
	tied.AddScoreSummary(own, &ScoreSummary{Score: own.Score}, false)
	assert.Equal(t, 1, tied.Ties)
	assert.Equal(t, 1+own.BonusRankingPoints, tied.RankingPoints)

	lost := &RankingFields{}
	lost.AddScoreSummary(own, &ScoreSummary{Score: own.Score + 1}, false)
	assert.Equal(t, 1, lost.Losses)
	assert.Equal(t, own.BonusRankingPoints, lost.RankingPoints)

	dq := &RankingFields{}
	dq.AddScoreSummary(own, opponent, true)
	assert.Equal(t, 1, dq.Disqualifications)
	assert.Equal(t, 0, dq.RankingPoints)
	assert.Empty(t, dq.Tiebreakers)
}
