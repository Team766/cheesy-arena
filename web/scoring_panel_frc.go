// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//go:build !custom

// The game-specific half of the scoring panel: the template it renders and the websocket commands
// that mutate a Score. The handler and websocket loop they plug into live in scoring_panel.go.

package web

import (
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/websocket"
	"github.com/mitchellh/mapstructure"
)

const scoringPanelTemplatePath = "templates/scoring_panel.html"

// handleScoringPanelGameCommand applies one game-specific scoring command and reports whether the
// score changed. Unknown commands are ignored.
func handleScoringPanelGameCommand(
	ws *websocket.Websocket, score *game.Score, command string, data any,
) bool {
	switch command {
	case "autoTower":
		args := struct {
			TeamPosition    int
			AutoTowerStatus int
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}

		if args.TeamPosition >= 1 && args.TeamPosition <= 3 && args.AutoTowerStatus >= 0 &&
			args.AutoTowerStatus <= 3 {
			score.AutoTowerStatuses[args.TeamPosition-1] = game.TowerStatus(args.AutoTowerStatus)
			return true
		}
	case "endgame":
		args := struct {
			TeamPosition       int
			EndgameTowerStatus int
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}

		if args.TeamPosition >= 1 && args.TeamPosition <= 3 && args.EndgameTowerStatus >= 0 &&
			args.EndgameTowerStatus <= 3 {
			score.EndgameTowerStatuses[args.TeamPosition-1] = game.TowerStatus(args.EndgameTowerStatus)
			return true
		}
	}
	return false
}
