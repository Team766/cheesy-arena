//go:build !custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"strconv"
)

// The CSV report keeps the raw RankingFields names it has always used, including the win/loss/tie
// record split across three columns.
func rankingCsvColumns() []rankingColumn {
	return []rankingColumn{
		{Header: "Rank", Value: func(r game.Ranking) string { return strconv.Itoa(r.Rank) }},
		{Header: "TeamId", Value: func(r game.Ranking) string { return strconv.Itoa(r.TeamId) }},
		{Header: "RankingPoints", Value: func(r game.Ranking) string { return strconv.Itoa(r.RankingPoints) }},
		{Header: "MatchPoints", Value: func(r game.Ranking) string { return strconv.Itoa(r.MatchPoints) }},
		{Header: "AutoFuelPoints", Value: func(r game.Ranking) string { return strconv.Itoa(r.AutoFuelPoints) }},
		{Header: "TowerPoints", Value: func(r game.Ranking) string { return strconv.Itoa(r.TowerPoints) }},
		{Header: "Wins", Value: func(r game.Ranking) string { return strconv.Itoa(r.Wins) }},
		{Header: "Losses", Value: func(r game.Ranking) string { return strconv.Itoa(r.Losses) }},
		{Header: "Ties", Value: func(r game.Ranking) string { return strconv.Itoa(r.Ties) }},
		{
			Header: "Disqualifications",
			Value:  func(r game.Ranking) string { return strconv.Itoa(r.Disqualifications) },
		},
		{Header: "Played", Value: func(r game.Ranking) string { return strconv.Itoa(r.Played) }},
	}
}

// The PDF report is laid out for the printed page, so it uses shorter headers, a combined W-L-T
// column, and hand-tuned widths that add up to 195mm.
func rankingPdfColumns() []rankingColumn {
	return []rankingColumn{
		{Header: "Rank", Width: 13, Value: func(r game.Ranking) string { return strconv.Itoa(r.Rank) }},
		{Header: "Team", Width: 20, Value: func(r game.Ranking) string { return strconv.Itoa(r.TeamId) }},
		{Header: "RP", Width: 24, Value: func(r game.Ranking) string { return strconv.Itoa(r.RankingPoints) }},
		{Header: "Match", Width: 24, Value: func(r game.Ranking) string { return strconv.Itoa(r.MatchPoints) }},
		{
			Header: "Auto Fuel",
			Width:  24,
			Value:  func(r game.Ranking) string { return strconv.Itoa(r.AutoFuelPoints) },
		},
		{Header: "Tower", Width: 24, Value: func(r game.Ranking) string { return strconv.Itoa(r.TowerPoints) }},
		{
			Header: "W-L-T",
			Width:  26,
			Value:  func(r game.Ranking) string { return fmt.Sprintf("%d-%d-%d", r.Wins, r.Losses, r.Ties) },
		},
		{Header: "DQ", Width: 20, Value: func(r game.Ranking) string { return strconv.Itoa(r.Disqualifications) }},
		{Header: "Played", Width: 20, Value: func(r game.Ranking) string { return strconv.Itoa(r.Played) }},
	}
}
