//go:build custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/websocket"
	"github.com/mitchellh/mapstructure"
	"io"
	"log"
	"net/http"
)

type ScoringPosition struct {
	Title    string
	Alliance string
}

var positionParameters = map[string]ScoringPosition{
	"red": {
		Title:    "Red",
		Alliance: "red",
	},
	"blue": {
		Title:    "Blue",
		Alliance: "blue",
	},
}

// Renders the scoring interface which enables input of scores in real-time.
func (web *Web) scoringPanelHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	position := r.PathValue("position")
	parameters, ok := positionParameters[position]
	if !ok {
		handleWebErr(w, fmt.Errorf("Invalid position '%s'.", position))
		return
	}

	scoringPanelTemplate := "templates/generated_scoring_panel.html"
	template, err := web.parseFiles(scoringPanelTemplate, "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}
	data := struct {
		*model.EventSettings
		PositionName string
		Position     ScoringPosition
	}{web.arena.EventSettings, position, parameters}
	err = template.ExecuteTemplate(w, "base_no_navbar", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// The websocket endpoint for the scoring interface client to send control commands and receive status updates.
func (web *Web) scoringPanelWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	position := r.PathValue("position")
	_, ok := positionParameters[position]
	if !ok {
		handleWebErr(w, fmt.Errorf("Invalid position '%s'.", position))
		return
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer closeWebsocket(ws)
	web.arena.ScoringPanelRegistry.RegisterPanel(position, ws)
	web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringStatusNotifier.Notify()
	defer web.arena.ScoringPanelRegistry.UnregisterPanel(position, ws)

	// Instruct panel to clear any local state in case this is a reconnect
	writeWebsocketMessage(ws, "resetLocalState", nil)

	// Subscribe the websocket to the notifiers whose messages will be passed on to the client, in a separate goroutine.
	go ws.HandleNotifiers(
		web.arena.MatchLoadNotifier,
		web.arena.MatchTimeNotifier,
		web.arena.RealtimeScoreNotifier,
		web.arena.ReloadDisplaysNotifier,
	)

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		command, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Println(err)
			return
		}

		var score *game.Score
		if position == "red" {
			score = &web.arena.RedRealtimeScore.CurrentScore
		} else {
			score = &web.arena.BlueRealtimeScore.CurrentScore
		}
		scoreChanged := false

		if command == "commitMatch" {
			if web.arena.MatchState != field.PostMatch {
				writeWebsocketError(ws, "Cannot commit score: Match is not over.")
				continue
			}
			web.arena.ScoringPanelRegistry.SetScoreCommitted(position, ws)
			web.arena.ScoringStatusNotifier.Notify()
		} else if command == "addFoul" {
			args := struct {
				Alliance string
				IsMajor  bool
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}

			// Add the foul to the correct alliance's list.
			foul := game.Foul{FoulId: web.arena.NextFoulId, IsMajor: args.IsMajor}
			web.arena.NextFoulId++
			if args.Alliance == "red" {
				web.arena.RedRealtimeScore.CurrentScore.Fouls =
					append(web.arena.RedRealtimeScore.CurrentScore.Fouls, foul)
			} else {
				web.arena.BlueRealtimeScore.CurrentScore.Fouls =
					append(web.arena.BlueRealtimeScore.CurrentScore.Fouls, foul)
			}
			web.arena.RealtimeScoreNotifier.Notify()
		} else if command == "adjustCount" {
			// general purpose command for adjusting the count for a specific gamepiece
			// the game-specific logic is handled within Score.AdjustCount
			args := struct {
				Id    string
				Phase string
				Delta int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			var phase game.Phase
			switch args.Phase {
			case "auto":
				phase = game.PhaseAuto
			case "endgame":
				phase = game.PhaseEndgame
			default:
				phase = game.PhaseTeleop
			}
			if score.AdjustCount(args.Id, phase, args.Delta) {
				scoreChanged = true
			}
		} else if command == "setStatus" {
			// general purpose command for adjusting the boolean status of a robot
			// the game-specific logic is handled within Score.SetBoolStatus
			args := struct {
				Id         string
				RobotIndex int
				Value      bool
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			if score.SetBoolStatus(args.Id, args.RobotIndex, args.Value) {
				scoreChanged = true
			}
		} else if command == "setEnumStatus" {
			// Same as setStatus, but Value is replaced by ValueId (a custom_game.yaml status value id,
			// e.g. "full") for statuses declared with an enum `values` list.
			args := struct {
				Id         string
				RobotIndex int
				ValueId    string
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			if score.SetEnumStatus(args.Id, args.RobotIndex, args.ValueId) {
				scoreChanged = true
			}
		} else if command == "cycleEnumStatus" {
			// Advances an enum status to its next value, wrapping around — the scoring panel UI
			// uses one button per robot for enum statuses, cycling through values on each click.
			args := struct {
				Id         string
				RobotIndex int
			}{}
			err = mapstructure.Decode(data, &args)
			if err != nil {
				writeWebsocketError(ws, err.Error())
				continue
			}
			if score.CycleEnumStatus(args.Id, args.RobotIndex) {
				scoreChanged = true
			}
		}

		if scoreChanged {
			web.arena.RealtimeScoreNotifier.Notify()
		}
	}
}
