// Copyright 2017 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
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
	matchResult.RedScore.AutoGp1Level1Count = 1 // 5 points
	database.CreateMatchResult(matchResult)

	matches, _ := database.GetMatchesByType(model.Qualification, false)
	t.Logf("DEBUG: Matches count: %d", len(matches))
	if len(matches) > 0 {
		t.Logf("DEBUG: Match 1 ID: %d, Status: %v, Complete: %t", matches[0].Id, matches[0].Status, matches[0].IsComplete())
		mr, err := database.GetMatchResultForMatch(matches[0].Id)
		t.Logf("DEBUG: GetMatchResultForMatch err: %v, found: %t", err, mr != nil)
	}

	updatedRankings, err := CalculateRankings(database, false)
	t.Logf("DEBUG: updatedRankings count: %d, err: %v", len(updatedRankings), err)
	assert.Nil(t, err)
	assert.Len(t, updatedRankings, 6)

	// The 3 red teams (1, 3, 5) should have 3 RP.
	for i := 0; i < 3; i++ {
		assert.Contains(t, []int{1, 3, 5}, updatedRankings[i].TeamId)
		assert.Equal(t, i+1, updatedRankings[i].Rank)
		assert.Equal(t, 3, updatedRankings[i].RankingPoints)
	}

	// The 3 blue teams (2, 4, 6) should have 0 RP.
	for i := 3; i < 6; i++ {
		assert.Contains(t, []int{2, 4, 6}, updatedRankings[i].TeamId)
		assert.Equal(t, i+1, updatedRankings[i].Rank)
		assert.Equal(t, 0, updatedRankings[i].RankingPoints)
	}
}
