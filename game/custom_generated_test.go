// Copyright 2026 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//go:build custom

package game

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScoreEquals(t *testing.T) {
	s1 := &Score{
		AutoGp1Level1Count: 1,
		Fouls:              []Foul{{FoulId: 1, IsMajor: true}},
	}
	s2 := &Score{
		AutoGp1Level1Count: 1,
		Fouls:              []Foul{{FoulId: 1, IsMajor: true}},
	}
	s3 := &Score{
		AutoGp1Level1Count: 2,
		Fouls:              []Foul{{FoulId: 1, IsMajor: true}},
	}
	s4 := &Score{
		AutoGp1Level1Count: 1,
		Fouls:              []Foul{{FoulId: 2, IsMajor: true}},
	}

	assert.True(t, s1.Equals(s2))
	assert.False(t, s1.Equals(s3))
	assert.False(t, s1.Equals(s4))
}

func TestScoreSummarizeAutoPortion(t *testing.T) {
	score := &Score{
		AutoGp1Level1Count:   1,                           // 5 pts auto
		TeleopGp1Level1Count: 1,                           // 3 pts teleop
		LeaveStatuses:        [3]bool{true, false, false}, // 3 pts auto
	}
	opponent := &Score{
		Fouls: []Foul{{FoulId: 1, IsMajor: false}}, // 2 pts foul points to score
	}

	summary := score.Summarize(opponent)

	// Gp1Level1Points = 5 + 3 = 8
	assert.Equal(t, 8, summary.Gp1Level1Points)
	// AutoPoints = 5 (gp1_level1 auto) + 3 (leave) = 8
	assert.Equal(t, 8, summary.AutoPoints)
	// TeleopPoints = 3 (gp1_level1 teleop) = 3
	assert.Equal(t, 3, summary.TeleopPoints)
	// MatchPoints = 8 + 3 = 11
	assert.Equal(t, 11, summary.MatchPoints)
	// FoulPoints = 2 (minor foul points in game.yaml)
	assert.Equal(t, 2, summary.FoulPoints)
	// Score = 11 + 2 = 13
	assert.Equal(t, 13, summary.Score)
}

func TestDetermineMatchStatusTiebreaker(t *testing.T) {
	// Tied score, Red wins on auto points
	red := &ScoreSummary{
		Score:       10,
		AutoPoints:  6,
		MatchPoints: 10,
	}
	blue := &ScoreSummary{
		Score:       10,
		AutoPoints:  4,
		MatchPoints: 10,
	}

	// Without tiebreakers: returns TieMatch
	status, label := DetermineMatchStatus(red, blue, false)
	assert.Equal(t, TieMatch, status)
	assert.Equal(t, "", label)

	// With tiebreakers: returns RedWonMatch with auto points label
	status, label = DetermineMatchStatus(red, blue, true)
	assert.Equal(t, RedWonMatch, status)
	assert.Equal(t, "TIEBREAK: AUTO POINTS", label)

	// Tied score and auto points, Blue wins on total points (MatchPoints)
	red.AutoPoints = 4
	red.MatchPoints = 8
	blue.MatchPoints = 10
	status, label = DetermineMatchStatus(red, blue, true)
	assert.Equal(t, BlueWonMatch, status)
	assert.Equal(t, "TIEBREAK: TOTAL POINTS", label)
}

func TestAddScoreSummary(t *testing.T) {
	fields := &RankingFields{}
	own := &ScoreSummary{
		Score:              15,
		MatchPoints:        15,
		AutoPoints:         8,
		BonusRankingPoints: 1,
	}
	opponent := &ScoreSummary{
		Score: 10,
	}

	fields.AddScoreSummary(own, opponent, false)

	assert.Equal(t, 1, fields.Played)
	assert.Equal(t, 1, fields.Wins)
	assert.Equal(t, 4, fields.RankingPoints) // 3 for win + 1 bonus RP
	assert.Equal(t, 15, fields.MatchPoints)
	assert.Equal(t, 8, fields.AutoPoints)
}

func TestRankingsLess(t *testing.T) {
	rankings := Rankings{
		{
			TeamId: 1,
			RankingFields: RankingFields{
				RankingPoints: 10,
				MatchPoints:   200,
				AutoPoints:    80,
				Played:        10,
			},
		},
		{
			TeamId: 2,
			RankingFields: RankingFields{
				RankingPoints: 10,
				MatchPoints:   200,
				AutoPoints:    60,
				Played:        10,
			},
		},
		{
			TeamId: 3,
			RankingFields: RankingFields{
				RankingPoints: 8,
				MatchPoints:   300,
				AutoPoints:    100,
				Played:        10,
			},
		},
	}

	// 1st ranking points tie: Team 1 (AutoPoints 80) should sort before Team 2 (AutoPoints 60).
	// Team 3 has fewer ranking points, should sort last.
	assert.True(t, rankings.Less(0, 1)) // Team 1 vs Team 2
	assert.True(t, rankings.Less(0, 2)) // Team 1 vs Team 3
	assert.True(t, rankings.Less(1, 2)) // Team 2 vs Team 3
}

func TestGetAllRulesCustom(t *testing.T) {
	rules := GetAllRules()
	assert.NotEmpty(t, rules)
	for id, rule := range rules {
		assert.NotNil(t, GetRuleById(id))
		assert.Equal(t, rule, GetRuleById(id))
	}
}

func TestFoulPointValueCustom(t *testing.T) {
	fMajor := Foul{IsMajor: true}
	fMinor := Foul{IsMajor: false}

	assert.Equal(t, MajorFoulPoints, fMajor.PointValue())
	assert.Equal(t, MinorFoulPoints, fMinor.PointValue())
}
