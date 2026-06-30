//go:build custom

package tournament

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestCalculateRankingsCustom(t *testing.T) {
	randomizer := rand.New(rand.NewSource(1))
	game.RankingRandomFloat64 = randomizer.Float64
	database := setupTestDb(t)

	// Create 6 teams and 1 match
	for i := 1; i <= 6; i++ {
		assert.Nil(t, database.CreateTeam(&model.Team{Id: i, Nickname: fmt.Sprintf("Team %d", i)}))
	}

	match := &model.Match{
		Type:      model.Qualification,
		TypeOrder: 1,
		Time:      time.Unix(0, 0),
		Red1:      1,
		Red2:      3,
		Red3:      5,
		Blue1:     2,
		Blue2:     4,
		Blue3:     6,
		Status:    game.RedWonMatch,
	}
	assert.Nil(t, database.CreateMatch(match))

	// Red won match
	matchResult := &model.MatchResult{
		MatchId:    match.Id,
		PlayNumber: 1,
		RedScore: &game.Score{
			PlayoffDq: false,
		},
		BlueScore: &game.Score{
			PlayoffDq: false,
		},
	}
	// Give Red some points
	matchResult.RedScore.AutoStructure1Level1Count = 3 // 9 points, and qualifies for AutonRP (> 2 pieces on Structure 1)
	database.CreateMatchResult(matchResult)

	updatedRankings, err := CalculateRankings(database, false)
	assert.Nil(t, err)
	assert.Len(t, updatedRankings, 6)

	// The 3 red teams (1, 3, 5) should have 3 RP for the win plus 1 bonus RP for the AutonRP
	// (AutoStructure1Level1Count + AutoStructure1Level2Count > 2).
	for i := 0; i < 3; i++ {
		assert.Contains(t, []int{1, 3, 5}, updatedRankings[i].TeamId)
		assert.Equal(t, i+1, updatedRankings[i].Rank)
		assert.Equal(t, 4, updatedRankings[i].RankingPoints)
	}

	// The 3 blue teams (2, 4, 6) should have 0 RP.
	for i := 3; i < 6; i++ {
		assert.Contains(t, []int{2, 4, 6}, updatedRankings[i].TeamId)
		assert.Equal(t, i+1, updatedRankings[i].Rank)
		assert.Equal(t, 0, updatedRankings[i].RankingPoints)
	}
}
