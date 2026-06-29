//go:build custom

package game

func TestScore1() *Score {
	fouls := []Foul{
		{1, true, 25, 16},
		{2, false, 1868, 13},
		{3, false, 1868, 13},
		{4, true, 25, 15},
		{5, true, 25, 15},
		{6, true, 25, 15},
		{7, true, 25, 15},
	}
	return &Score{
		Fouls:     fouls,
		PlayoffDq: false,
	}
}

func TestScore2() *Score {
	return &Score{
		Fouls:     []Foul{},
		PlayoffDq: false,
	}
}

func TestRanking1() *Ranking {
	return &Ranking{
		TeamId: 254,
		Rank:   1,
		RankingFields: RankingFields{
			RankingPoints: 20,
			MatchPoints:   10,
			AutoPoints:    10,
			Random:        0.254,
			Wins:          3,
			Losses:        2,
			Ties:          1,
			Played:        10,
		},
	}
}

func TestRanking2() *Ranking {
	return &Ranking{
		TeamId: 1114,
		Rank:   2,
		RankingFields: RankingFields{
			RankingPoints: 18,
			MatchPoints:   5,
			AutoPoints:    5,
			Random:        0.1114,
			Wins:          1,
			Losses:        3,
			Ties:          2,
			Played:        10,
		},
	}
}
