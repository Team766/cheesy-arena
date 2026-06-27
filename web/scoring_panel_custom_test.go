// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//go:build custom

package web

import (
	"github.com/Team254/cheesy-arena/field"
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

	// Send some count adjustment commands.
	adjustData := struct {
		Id    string
		Phase string
		Delta int
	}{}
	assert.Equal(t, 0, web.arena.RedRealtimeScore.CurrentScore.AutoGp1Level1Count)
	assert.Equal(t, 0, web.arena.RedRealtimeScore.CurrentScore.TeleopGp1Level1Count)
	assert.Equal(t, 0, web.arena.BlueRealtimeScore.CurrentScore.AutoGp1Level1Count)
	assert.Equal(t, 0, web.arena.BlueRealtimeScore.CurrentScore.TeleopGp1Level1Count)

	adjustData.Id = "gp1_level1"
	adjustData.Phase = "auto"
	adjustData.Delta = 2
	redWs.Write("adjustCount", adjustData)

	adjustData.Id = "gp1_level1"
	adjustData.Phase = "teleop"
	adjustData.Delta = 3
	blueWs.Write("adjustCount", adjustData)

	adjustData.Id = "gp2"
	adjustData.Phase = "teleop"
	adjustData.Delta = 1
	redWs.Write("adjustCount", adjustData)

	for i := 0; i < 3; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}

	assert.Equal(t, 2, web.arena.RedRealtimeScore.CurrentScore.AutoGp1Level1Count)
	assert.Equal(t, 3, web.arena.BlueRealtimeScore.CurrentScore.TeleopGp1Level1Count)
	assert.Equal(t, 1, web.arena.RedRealtimeScore.CurrentScore.TeleopGp2Count)

	// Send status commands.
	statusData := struct {
		Id         string
		RobotIndex int
		Value      bool
	}{}
	assert.Equal(t, [3]bool{false, false, false}, web.arena.RedRealtimeScore.CurrentScore.LeaveStatuses)
	assert.Equal(t, [3]bool{false, false, false}, web.arena.BlueRealtimeScore.CurrentScore.ParkStatuses)

	statusData.Id = "leave"
	statusData.RobotIndex = 0
	statusData.Value = true
	redWs.Write("setStatus", statusData)

	statusData.Id = "park"
	statusData.RobotIndex = 2
	statusData.Value = true
	blueWs.Write("setStatus", statusData)

	for i := 0; i < 2; i++ {
		readWebsocketType(t, redWs, "realtimeScore")
		readWebsocketType(t, blueWs, "realtimeScore")
	}

	assert.Equal(t, [3]bool{true, false, false}, web.arena.RedRealtimeScore.CurrentScore.LeaveStatuses)
	assert.Equal(t, [3]bool{false, false, true}, web.arena.BlueRealtimeScore.CurrentScore.ParkStatuses)

	// Add a couple of fouls.
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
