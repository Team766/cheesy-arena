//go:build custom

package game

import (
	"math/rand"
)

type RankingFields struct {
	RankingPoints     int            `json:"ranking_points"`
	Tiebreakers       map[string]int `json:"tiebreakers"`
	Random            float64        `json:"random"`
	Wins              int            `json:"wins"`
	Losses            int            `json:"losses"`
	Ties              int            `json:"ties"`
	Disqualifications int            `json:"disqualifications"`
	Played            int            `json:"played"`
}

type Ranking struct {
	TeamId       int `db:"id,manual"`
	Rank         int
	PreviousRank int
	RankingFields
}

type Rankings []Ranking

var RankingRandomFloat64 = rand.Float64

func (fields *RankingFields) AddScoreSummary(ownScore *ScoreSummary, opponentScore *ScoreSummary, disqualified bool) {
	fields.Played += 1
	fields.Random = RankingRandomFloat64()

	if fields.Tiebreakers == nil {
		fields.Tiebreakers = make(map[string]int)
	}

	if disqualified {
		fields.Disqualifications += 1
		return
	}

	if ownScore.Score > opponentScore.Score {
		fields.RankingPoints += 3
		fields.Wins += 1
	} else if ownScore.Score == opponentScore.Score {
		fields.RankingPoints += 1
		fields.Ties += 1
	} else {
		fields.Losses += 1
	}
	fields.RankingPoints += ownScore.BonusRankingPoints

	cfg := GetActiveConfig()
	if cfg != nil {
		for _, tb := range cfg.RankingTiebreakers {
			fields.Tiebreakers[tb.Metric] += ownScore.GetMetric(tb.Metric)
		}
	}
}

func (rankings Rankings) Len() int {
	return len(rankings)
}

func (rankings Rankings) Less(i, j int) bool {
	a := rankings[i]
	b := rankings[j]

	if a.RankingPoints*b.Played != b.RankingPoints*a.Played {
		return a.RankingPoints*b.Played > b.RankingPoints*a.Played
	}

	cfg := GetActiveConfig()
	if cfg != nil {
		for _, tb := range cfg.RankingTiebreakers {
			aMetric := a.Tiebreakers[tb.Metric]
			bMetric := b.Tiebreakers[tb.Metric]
			if aMetric*b.Played != bMetric*a.Played {
				return aMetric*b.Played > bMetric*a.Played
			}
		}
	}

	return a.Random > b.Random
}

func (rankings Rankings) Swap(i, j int) {
	rankings[i], rankings[j] = rankings[j], rankings[i]
}
