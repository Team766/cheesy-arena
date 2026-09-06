//go:build custom

package tournament

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

// TestCalculateRankingsCustom drives the whole custom pipeline end to end: a match result whose
// scores are built by the config-driven mutators, through Summarize, AddScoreSummary, the
// config-driven Rankings sort, and into the database. Everything it asserts is derived from the
// active fixture config, so it holds for any valid game.
func TestCalculateRankingsCustom(t *testing.T) {
	randomizer := rand.New(rand.NewSource(1))
	game.RankingRandomFloat64 = randomizer.Float64
	database := setupTestDb(t)
	cfg := game.LoadFixtureConfig(t)

	// Score the first count in the fixture, in its first phase, for the red alliance only.
	count := cfg.ScoringCounts[0]
	phase, ok := game.PhaseFromString(count.Phases[0].Phase)
	assert.True(t, ok)
	pointsEach := count.Phases[0].Points

	match := model.Match{
		Type: model.Qualification, ShortName: "Q1", Status: game.RedWonMatch,
		Red1: 1, Red2: 2, Red3: 3, Blue1: 4, Blue2: 5, Blue3: 6,
	}
	assert.Nil(t, database.CreateMatch(&match))

	redScore := new(game.Score)
	assert.True(t, redScore.AdjustCount(count.ID, phase, 4))
	matchResult := model.NewMatchResult()
	matchResult.MatchId = match.Id
	matchResult.RedScore = redScore
	matchResult.BlueScore = new(game.Score)
	assert.Nil(t, database.CreateMatchResult(matchResult))

	rankings, err := CalculateRankings(database, false)
	assert.Nil(t, err)
	if !assert.Equal(t, 6, len(rankings)) {
		return
	}

	stored, err := database.GetAllRankings()
	assert.Nil(t, err)
	assert.Equal(t, rankings, stored)

	byTeam := make(map[int]game.Ranking)
	for _, ranking := range rankings {
		byTeam[ranking.TeamId] = ranking
	}

	expectedSummary := redScore.Summarize(new(game.Score))
	assert.Equal(t, 4*pointsEach, expectedSummary.Score)

	// The three winners lead, each with 3 RP for the win plus whatever bonus RPs the config's
	// logic functions awarded for this score.
	expectedWinnerRp := 3 + expectedSummary.BonusRankingPoints
	for _, teamId := range []int{1, 2, 3} {
		ranking := byTeam[teamId]
		assert.Equal(t, expectedWinnerRp, ranking.RankingPoints, "team %d", teamId)
		assert.Equal(t, 1, ranking.Wins, "team %d", teamId)
		assert.Equal(t, 0, ranking.Losses, "team %d", teamId)
		assert.Equal(t, 1, ranking.Played, "team %d", teamId)
		assert.LessOrEqual(t, ranking.Rank, 3, "team %d should be ranked above the losers", teamId)

		// The ranking tiebreaker columns hold exactly the metrics the config names.
		assert.Len(t, ranking.Tiebreakers, len(cfg.RankingTiebreakers))
		for _, tb := range cfg.RankingTiebreakers {
			assert.Equal(
				t, expectedSummary.GetMetric(tb.Metric), ranking.Tiebreakers[tb.Metric], tb.Metric,
			)
		}
	}

	blueSummary := new(game.Score).Summarize(redScore)
	for _, teamId := range []int{4, 5, 6} {
		ranking := byTeam[teamId]
		assert.Equal(t, blueSummary.BonusRankingPoints, ranking.RankingPoints, "team %d", teamId)
		assert.Equal(t, 0, ranking.Wins, "team %d", teamId)
		assert.Equal(t, 1, ranking.Losses, "team %d", teamId)
		assert.Greater(t, ranking.Rank, 3, "team %d should be ranked below the winners", teamId)
	}
}

// TestCalculateRankingsCustomTie checks the tie path, which awards one ranking point to everyone.
func TestCalculateRankingsCustomTie(t *testing.T) {
	randomizer := rand.New(rand.NewSource(2))
	game.RankingRandomFloat64 = randomizer.Float64
	database := setupTestDb(t)
	cfg := game.LoadFixtureConfig(t)

	count := cfg.ScoringCounts[0]
	phase, _ := game.PhaseFromString(count.Phases[0].Phase)

	match := model.Match{
		Type: model.Qualification, ShortName: "Q1", Status: game.TieMatch,
		Red1: 1, Red2: 2, Red3: 3, Blue1: 4, Blue2: 5, Blue3: 6,
	}
	assert.Nil(t, database.CreateMatch(&match))

	redScore := new(game.Score)
	blueScore := new(game.Score)
	assert.True(t, redScore.AdjustCount(count.ID, phase, 2))
	assert.True(t, blueScore.AdjustCount(count.ID, phase, 2))

	matchResult := model.NewMatchResult()
	matchResult.MatchId = match.Id
	matchResult.RedScore = redScore
	matchResult.BlueScore = blueScore
	assert.Nil(t, database.CreateMatchResult(matchResult))

	rankings, err := CalculateRankings(database, false)
	assert.Nil(t, err)
	if !assert.Equal(t, 6, len(rankings)) {
		return
	}

	expectedRp := 1 + redScore.Summarize(blueScore).BonusRankingPoints
	for _, ranking := range rankings {
		assert.Equal(t, 1, ranking.Ties, "team %d", ranking.TeamId)
		assert.Equal(t, expectedRp, ranking.RankingPoints, "team %d", ranking.TeamId)
	}
}
