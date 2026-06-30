//go:build custom

package web

import (
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/websocket"
	gorillawebsocket "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestScoringPanelCustom(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/panels/scoring/invalidposition")
	assert.Equal(t, 500, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Invalid position")
	recorder = web.getHttpResponse("/panels/scoring/red")
	assert.Equal(t, 200, recorder.Code)
	recorder = web.getHttpResponse("/panels/scoring/blue")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Custom Scoring Panel")
}

func TestScoringPanelWebsocketCustom(t *testing.T) {
	web := setupTestWeb(t)

	server, wsUrl := web.startTestServer()
	defer server.Close()
	_, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blorpy/websocket", nil)
	assert.NotNil(t, err)
	redConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/red/websocket", nil)
	assert.Nil(t, err)
	defer redConn.Close()
	redWs := websocket.NewTestWebsocket(redConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumPanels("blue"))
	blueConn, _, err := gorillawebsocket.DefaultDialer.Dial(wsUrl+"/panels/scoring/blue/websocket", nil)
	assert.Nil(t, err)
	defer blueConn.Close()
	blueWs := websocket.NewTestWebsocket(blueConn)
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("red"))
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumPanels("blue"))

	// Should get a few status updates right after connection.
	readWebsocketType(t, redWs, "resetLocalState")
	readWebsocketType(t, redWs, "matchLoad")
	readWebsocketType(t, redWs, "matchTime")
	readWebsocketType(t, redWs, "realtimeScore")
	readWebsocketType(t, blueWs, "resetLocalState")
	readWebsocketType(t, blueWs, "matchLoad")
	readWebsocketType(t, blueWs, "matchTime")
	readWebsocketType(t, blueWs, "realtimeScore")

	// adjustCount / setStatus dispatch is exercised generically here: the per-element id→field routing
	// and point math are owned by the generated Score tests (generated_score*_test.go, regenerated per
	// custom_game.yaml), so this test stays config-agnostic. An unknown id is a graceful no-op — the
	// handler only broadcasts on a real change — which we confirm below (no points leak into the score).
	redWs.Write("adjustCount", struct {
		Id    string
		Phase string
		Delta int
	}{Id: "__nonexistent__", Phase: "auto", Delta: 5})
	redWs.Write("setStatus", struct {
		Id         string
		RobotIndex int
		Value      bool
	}{Id: "__nonexistent__", RobotIndex: 0, Value: true})

	// Add a couple of fouls — a websocket command that always changes the score, exercising the full
	// cmd → handler → score → RealtimeScoreNotifier → broadcast pipeline config-agnostically.
	foulData := struct {
		Alliance string
		IsMajor  bool
	}{Alliance: "red", IsMajor: true}
	redWs.Write("addFoul", foulData)
	foulData = struct {
		Alliance string
		IsMajor  bool
	}{Alliance: "blue", IsMajor: false}
	blueWs.Write("addFoul", foulData)
	for i := 0; i < 2; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}
	assert.Equal(t, 1, len(web.arena.RedRealtimeScore.CurrentScore.Fouls))
	assert.Equal(t, true, web.arena.RedRealtimeScore.CurrentScore.Fouls[0].IsMajor)
	assert.Equal(t, 1, len(web.arena.BlueRealtimeScore.CurrentScore.Fouls))
	assert.Equal(t, false, web.arena.BlueRealtimeScore.CurrentScore.Fouls[0].IsMajor)

	// The earlier unknown-id adjustCount/setStatus were no-ops: no element/status points entered the score.
	assert.Equal(t, 0, web.arena.RedRealtimeScore.CurrentScore.Summarize(&game.Score{}).MatchPoints)

	// Test committing logic.
	redWs.Write("commitMatch", nil)
	readWebsocketType(t, redWs, "error")
	blueWs.Write("commitMatch", nil)
	readWebsocketType(t, blueWs, "error")
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red"))
	assert.Equal(t, 0, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("blue"))
	web.arena.MatchState = field.PostMatch
	redWs.Write("commitMatch", nil)
	blueWs.Write("commitMatch", nil)
	time.Sleep(time.Millisecond * 10) // Allow some time for the commands to be processed.
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("red"))
	assert.Equal(t, 1, web.arena.ScoringPanelRegistry.GetNumScoreCommitted("blue"))
}
