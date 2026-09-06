//go:build custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"strconv"
)

// A custom game's ranking columns are the fixed ones plus one per configured ranking tiebreaker,
// labelled with that metric's display name. The CSV and PDF reports use the same set.
func rankingReportColumns() []rankingColumn {
	columns := []rankingColumn{
		{Header: "Rank", Value: func(r game.Ranking) string { return strconv.Itoa(r.Rank) }},
		{Header: "Team", Value: func(r game.Ranking) string { return strconv.Itoa(r.TeamId) }},
		{Header: "RP", Value: func(r game.Ranking) string { return strconv.Itoa(r.RankingPoints) }},
	}

	if cfg := game.GetActiveConfig(); cfg != nil {
		for _, tiebreaker := range cfg.RankingTiebreakers {
			metric := tiebreaker.Metric
			columns = append(
				columns,
				rankingColumn{
					Header: cfg.MetricLabel(metric),
					Value:  func(r game.Ranking) string { return strconv.Itoa(r.Tiebreakers[metric]) },
				},
			)
		}
	}

	columns = append(
		columns,
		rankingColumn{
			Header: "W-L-T",
			Value:  func(r game.Ranking) string { return fmt.Sprintf("%d-%d-%d", r.Wins, r.Losses, r.Ties) },
		},
		rankingColumn{Header: "DQ", Value: func(r game.Ranking) string { return strconv.Itoa(r.Disqualifications) }},
		rankingColumn{Header: "Played", Value: func(r game.Ranking) string { return strconv.Itoa(r.Played) }},
	)

	// The number of columns depends on the config, so the printed page is divided evenly rather
	// than using hand-tuned widths.
	width := 195.0 / float64(len(columns))
	for i := range columns {
		columns[i].Width = width
	}
	return columns
}

func rankingCsvColumns() []rankingColumn {
	return rankingReportColumns()
}

func rankingPdfColumns() []rankingColumn {
	return rankingReportColumns()
}
