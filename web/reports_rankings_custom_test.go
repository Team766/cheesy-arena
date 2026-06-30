//go:build custom

package web

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRankingsCsvReport(t *testing.T) {
	web := setupTestWeb(t)

	ranking1 := game.TestRanking2()
	ranking2 := game.TestRanking1()
	web.arena.Database.CreateRanking(ranking1)
	web.arena.Database.CreateRanking(ranking2)

	recorder := web.getHttpResponse("/reports/csv/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "text/plain", recorder.Header()["Content-Type"][0])
	expectedBody := "Rank,TeamId,RankingPoints,MatchPoints,AutoPoints,Wins,Losses,Ties,Disqualifications,Played\n1,254,20,10,10,3,2,1,0,10\n2,1114,18,5,5,1,3,2,0,10\n"
	assert.Equal(t, expectedBody, recorder.Body.String())
}

func TestRankingsPdfReport(t *testing.T) {
	web := setupTestWeb(t)

	ranking1 := game.TestRanking2()
	ranking2 := game.TestRanking1()
	web.arena.Database.CreateRanking(ranking1)
	web.arena.Database.CreateRanking(ranking2)

	recorder := web.getHttpResponse("/reports/pdf/rankings")
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "application/pdf", recorder.Header()["Content-Type"][0])
}
