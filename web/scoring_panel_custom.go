//go:build custom

// The game-specific half of the scoring panel for a custom game: the template it renders and the
// four generic websocket commands that mutate a Score. All of the game knowledge lives in the YAML
// config and is enforced by the Score mutators, so these handlers only decode arguments. The
// handler and websocket loop they plug into live in scoring_panel.go.

package web

import (
	"strings"

	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/websocket"
	"github.com/mitchellh/mapstructure"
)

const scoringPanelTemplatePath = "templates/custom_scoring_panel.html.tmpl"

// Custom games add a near and a far scoring panel per alliance. An element whose config carries
// `scorer: near` or `scorer: far` appears only on that panel; the plain red/blue panels still show
// everything, so an event can run with one or two scorers per alliance.
func init() {
	for _, alliance := range []string{"red", "blue"} {
		for _, scorer := range []string{game.ScorerNear, game.ScorerFar} {
			positionParameters[alliance+"_"+scorer] = ScoringPosition{
				Title:    strings.Title(alliance) + " " + strings.Title(scorer),
				Alliance: alliance,
				Scorer:   scorer,
			}
		}
	}
}

// handleScoringPanelGameCommand applies one game-specific scoring command and reports whether the
// score changed. Unknown commands, and commands naming an element the config doesn't declare, are
// ignored.
func handleScoringPanelGameCommand(
	ws *websocket.Websocket, score *game.Score, command string, data any,
) bool {
	switch command {
	case "adjustCount":
		// General-purpose command for adjusting the count for a specific game piece; the
		// game-specific validation is handled within Score.AdjustCount.
		args := struct {
			Id    string
			Phase string
			Delta int
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}
		phase, ok := game.PhaseFromString(args.Phase)
		if !ok {
			writeWebsocketError(ws, "Invalid phase: "+args.Phase)
			return false
		}
		return score.AdjustCount(args.Id, phase, args.Delta)
	case "setStatus":
		// General-purpose command for adjusting the boolean status of a robot; the game-specific
		// validation is handled within Score.SetBoolStatus.
		args := struct {
			Id         string
			RobotIndex int
			Value      bool
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}
		return score.SetBoolStatus(args.Id, args.RobotIndex, args.Value)
	case "setEnumStatus":
		// Same as setStatus, but Value is replaced by ValueId (a custom_game.yaml status value id,
		// e.g. "full") for statuses declared with an enum `values` list.
		args := struct {
			Id         string
			RobotIndex int
			ValueId    string
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}
		return score.SetEnumStatusByID(args.Id, args.RobotIndex, args.ValueId)
	case "cycleEnumStatus":
		// Advances an enum status to its next value, wrapping around — the scoring panel UI uses
		// one button per robot for enum statuses, cycling through values on each click.
		args := struct {
			Id         string
			RobotIndex int
		}{}
		if err := mapstructure.Decode(data, &args); err != nil {
			writeWebsocketError(ws, err.Error())
			return false
		}
		return score.CycleEnumStatus(args.Id, args.RobotIndex)
	}
	return false
}
