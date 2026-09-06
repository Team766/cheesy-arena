//go:build custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

// The rankings report columns are derived from the active config's ranking_tiebreakers, so this test
// derives its expectations the same way rather than typing in the shipped game's metrics.
func TestRankingsCsvReportCustom(t *testing.T) {
	web := setupTestWeb(t)
	cfg := game.GetActiveConfig()
	if !assert.NotNil(t, cfg, "the test arena should have activated the fixture config") {
		return
	}
	assert.NotEmpty(t, cfg.RankingTiebreakers)

	ranking1 := game.TestRanking2()
	ranking2 := game.TestRanking1()
	for i, ranking := range []*game.Ranking{ranking1, ranking2} {
		ranking.Tiebreakers = make(map[string]int)
		for j, tb := range cfg.RankingTiebreakers {
			ranking.Tiebreakers[tb.Metric] = 100*(i+1) + j
		}
		assert.Nil(t, web.arena.Database.CreateRanking(ranking))
	}

	recorder := web.getHttpResponse("/reports/csv/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "text/plain", recorder.Header()["Content-Type"][0])

	lines := strings.Split(strings.TrimRight(recorder.Body.String(), "\n"), "\n")
	if !assert.Len(t, lines, 3) {
		return
	}

	// Header: the fixed columns with the configured tiebreakers' DISPLAY NAMES between them, not
	// their raw metric ids.
	var expectedHeader []string
	expectedHeader = append(expectedHeader, "Rank", "Team", "RP")
	for _, tb := range cfg.RankingTiebreakers {
		expectedHeader = append(expectedHeader, cfg.MetricLabel(tb.Metric))
	}
	expectedHeader = append(expectedHeader, "W-L-T", "DQ", "Played")
	assert.Equal(t, strings.Join(expectedHeader, ","), lines[0])

	// Rows come back in rank order, with one cell per configured tiebreaker.
	for i, ranking := range []*game.Ranking{ranking2, ranking1} {
		var expectedRow []string
		expectedRow = append(
			expectedRow,
			strconv.Itoa(ranking.Rank),
			strconv.Itoa(ranking.TeamId),
			strconv.Itoa(ranking.RankingPoints),
		)
		for _, tb := range cfg.RankingTiebreakers {
			expectedRow = append(expectedRow, strconv.Itoa(ranking.Tiebreakers[tb.Metric]))
		}
		expectedRow = append(
			expectedRow,
			fmt.Sprintf("%d-%d-%d", ranking.Wins, ranking.Losses, ranking.Ties),
			strconv.Itoa(ranking.Disqualifications),
			strconv.Itoa(ranking.Played),
		)
		assert.Equal(t, strings.Join(expectedRow, ","), lines[i+1])
		assert.Len(t, strings.Split(lines[i+1], ","), len(expectedHeader))
	}
}

func TestRankingsPdfReportCustom(t *testing.T) {
	web := setupTestWeb(t)

	assert.Nil(t, web.arena.Database.CreateRanking(game.TestRanking2()))
	assert.Nil(t, web.arena.Database.CreateRanking(game.TestRanking1()))

	// The PDF content can't be parsed here, so just check that a PDF comes back.
	recorder := web.getHttpResponse("/reports/pdf/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/pdf", recorder.Header()["Content-Type"][0])
}
