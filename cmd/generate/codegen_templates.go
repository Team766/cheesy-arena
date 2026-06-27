// Code generator module for emitting scoring panels, audience displays (HTML/JS), and their template assertions.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PieceGroup collapses every ScoringCount sharing a GamePiece id into one entry (DisplayName
// resolved from yamlData.GamePieces); a ScoringCount with no GamePiece set gets its own
// single-element group, keyed and displayed by its own id/display name.
type PieceGroup struct {
	ID          string
	DisplayName string
	ElementIDs  []string
}

func buildPieceGroups(yamlData *GameYAML) []PieceGroup {
	var pieceGroups []PieceGroup
	seenPieces := make(map[string]int)

	for _, sc := range yamlData.ScoringCounts {
		if sc.GamePiece != "" {
			gpName := sc.GamePiece
			for _, gp := range yamlData.GamePieces {
				if gp.ID == sc.GamePiece {
					gpName = gp.DisplayName
				}
			}
			if idx, ok := seenPieces[sc.GamePiece]; ok {
				pieceGroups[idx].ElementIDs = append(pieceGroups[idx].ElementIDs, sc.ID)
			} else {
				seenPieces[sc.GamePiece] = len(pieceGroups)
				pieceGroups = append(pieceGroups, PieceGroup{
					ID:          sc.GamePiece,
					DisplayName: gpName,
					ElementIDs:  []string{sc.ID},
				})
			}
		} else {
			pieceGroups = append(pieceGroups, PieceGroup{
				ID:          sc.ID,
				DisplayName: sc.DisplayName,
				ElementIDs:  []string{sc.ID},
			})
		}
	}

	return pieceGroups
}

func generateScoringPanelTemplate(yamlData *GameYAML, templatesDir string) error {
	filePath := filepath.Join(templatesDir, "generated_scoring_panel.html")

	var sb strings.Builder
	sb.WriteString(`{{define "title"}}Custom Scoring Panel{{end}}
{{define "head"}}
<link rel="manifest" href="/static/manifest/{{.PositionName}}_scoring.manifest">
<meta name="viewport" content="width=device-width, user-scalable=no">
<link href="/static/css/scoring_panel.css" rel="stylesheet">
{{end}}
{{define "body"}}
<header id="banner">
  <div class="screen-title">{{.Position.Title}} - <span id="matchName">&nbsp;</span></div>
</header>
<main>
  <div id="tower-controls" style="display: flex; flex-direction: column; gap: 20px; width: 100%; flex: none; height: auto;">
`)

	// 1. Auto Section
	sb.WriteString("    <section class=\"tower-section\" id=\"auto-section\" style=\"flex: none; height: auto;\">\n      <h1>Auto</h1>\n      <div style=\"display: flex; flex-direction: column; gap: 15px; width: 100%;\">\n")
	// Count controls in Auto
	for _, sc := range yamlData.ScoringCounts {
		if sc.Phase == "auto" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf(`        <div class="count-control" id="%s-auto" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="count-label" style="font-size: 1.2rem; font-weight: bold;">%s</span>
          <div style="display: flex; align-items: center; gap: 15px;">
            <button class="scoring-button" onclick="adjustCount('%s', 'auto', -1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">-</button>
            <span class="count-value" id="%s-auto-count" style="font-size: 1.5rem; min-width: 30px; text-align: center;">0</span>
            <button class="scoring-button" onclick="adjustCount('%s', 'auto', 1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">+</button>
          </div>
        </div>
`, sc.ID, sc.DisplayName, sc.ID, sc.ID, sc.ID))
		}
	}
	// Status controls in Auto
	for _, status := range yamlData.Statuses {
		if status.Phase == "auto" {
			if len(status.Values) >= 2 {
				// Enum status: multiple buttons per robot
				sb.WriteString(fmt.Sprintf(`        <div class="status-control" id="%s-status" style="display: flex; flex-direction: column; gap: 8px; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="status-label" style="font-size: 1.2rem; font-weight: bold; margin-bottom: 5px;">%s</span>
          <div style="display: flex; flex-direction: column; gap: 10px; width: 100%%;">
            {{range $i := seq 3}}
            <div class="robot-status-row team-{{$i}}" style="display: flex; align-items: center; gap: 10px;">
              <span class="team-num" style="font-size: 1.1rem; width: 40px; font-weight: bold;"></span>
              <div style="display: flex; gap: 5px; flex-grow: 1;">
`, status.ID, status.DisplayName))
				for _, val := range status.Values {
					sb.WriteString(fmt.Sprintf(`                <button class="scoring-button status-toggle" id="%s-{{add $i -1}}-%s" onclick="setEnumStatus('%s', {{add $i -1}}, '%s');" ontouchstart disabled style="flex-grow: 1; height: 40px;">%s</button>
`, status.ID, val.ID, status.ID, val.ID, val.DisplayName))
				}
				sb.WriteString("              </div>\n            </div>\n            {{end}}\n          </div>\n        </div>\n")
			} else {
				// Bool status
				sb.WriteString(fmt.Sprintf(`        <div class="status-control" id="%s-status" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="status-label" style="font-size: 1.2rem; font-weight: bold;">%s</span>
          <div style="display: flex; gap: 8px;">
            {{range $i := seq 3}}
            <div class="team-{{$i}}" style="display: flex; flex-direction: column; align-items: center; gap: 4px;">
              <span class="team-num" style="font-size: 0.85rem; font-weight: bold;"></span>
              <button class="scoring-button status-toggle" id="%s-{{add $i -1}}" onclick="toggleBoolStatus('%s', {{add $i -1}});" ontouchstart disabled style="width: 60px; height: 45px;"></button>
            </div>
            {{end}}
          </div>
        </div>
`, status.ID, status.DisplayName, status.ID, status.ID))
			}
		}
	}
	sb.WriteString("      </div>\n    </section>\n")

	// 2. Teleop Section
	hasTeleop := false
	for _, sc := range yamlData.ScoringCounts {
		if sc.Phase == "teleop" || sc.Phase == "both" {
			hasTeleop = true
			break
		}
	}
	if hasTeleop {
		sb.WriteString("    <section class=\"tower-section\" id=\"teleop-section\" style=\"flex: none; height: auto;\">\n      <h1>Teleop</h1>\n      <div style=\"display: flex; flex-direction: column; gap: 15px; width: 100%;\">\n")
		for _, sc := range yamlData.ScoringCounts {
			if sc.Phase == "teleop" || sc.Phase == "both" {
				sb.WriteString(fmt.Sprintf(`        <div class="count-control" id="%s-teleop" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="count-label" style="font-size: 1.2rem; font-weight: bold;">%s</span>
          <div style="display: flex; align-items: center; gap: 15px;">
            <button class="scoring-button" onclick="adjustCount('%s', 'teleop', -1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">-</button>
            <span class="count-value" id="%s-teleop-count" style="font-size: 1.5rem; min-width: 30px; text-align: center;">0</span>
            <button class="scoring-button" onclick="adjustCount('%s', 'teleop', 1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">+</button>
          </div>
        </div>
`, sc.ID, sc.DisplayName, sc.ID, sc.ID, sc.ID))
			}
		}
		sb.WriteString("      </div>\n    </section>\n")
	}

	// 3. Endgame Section
	hasEndgame := len(yamlData.EndgameCounts) > 0
	for _, status := range yamlData.Statuses {
		if status.Phase == "endgame" {
			hasEndgame = true
			break
		}
	}
	if hasEndgame {
		sb.WriteString("    <section class=\"tower-section\" id=\"endgame-section\" style=\"flex: none; height: auto;\">\n      <h1>Endgame</h1>\n      <div style=\"display: flex; flex-direction: column; gap: 15px; width: 100%;\">\n")
		for _, ec := range yamlData.EndgameCounts {
			sb.WriteString(fmt.Sprintf(`        <div class="count-control" id="%s-endgame" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="count-label" style="font-size: 1.2rem; font-weight: bold;">%s</span>
          <div style="display: flex; align-items: center; gap: 15px;">
            <button class="scoring-button" onclick="adjustCount('%s', 'endgame', -1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">-</button>
            <span class="count-value" id="%s-endgame-count" style="font-size: 1.5rem; min-width: 30px; text-align: center;">0</span>
            <button class="scoring-button" onclick="adjustCount('%s', 'endgame', 1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">+</button>
          </div>
        </div>
`, ec.ID, ec.DisplayName, ec.ID, ec.ID, ec.ID))
		}
		for _, status := range yamlData.Statuses {
			if status.Phase == "endgame" {
				if len(status.Values) >= 2 {
					// Enum status: multiple buttons per robot
					sb.WriteString(fmt.Sprintf(`        <div class="status-control" id="%s-status" style="display: flex; flex-direction: column; gap: 8px; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="status-label" style="font-size: 1.2rem; font-weight: bold; margin-bottom: 5px;">%s</span>
          <div style="display: flex; flex-direction: column; gap: 10px; width: 100%%;">
            {{range $i := seq 3}}
            <div class="robot-status-row team-{{$i}}" style="display: flex; align-items: center; gap: 10px;">
              <span class="team-num" style="font-size: 1.1rem; width: 40px; font-weight: bold;"></span>
              <div style="display: flex; gap: 5px; flex-grow: 1;">
`, status.ID, status.DisplayName))
					for _, val := range status.Values {
						sb.WriteString(fmt.Sprintf(`                <button class="scoring-button status-toggle" id="%s-{{add $i -1}}-%s" onclick="setEnumStatus('%s', {{add $i -1}}, '%s');" ontouchstart disabled style="flex-grow: 1; height: 40px;">%s</button>
`, status.ID, val.ID, status.ID, val.ID, val.DisplayName))
					}
					sb.WriteString("              </div>\n            </div>\n            {{end}}\n          </div>\n        </div>\n")
				} else {
					// Bool status
					sb.WriteString(fmt.Sprintf(`        <div class="status-control" id="%s-status" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">
          <span class="status-label" style="font-size: 1.2rem; font-weight: bold;">%s</span>
          <div style="display: flex; gap: 8px;">
            {{range $i := seq 3}}
            <div class="team-{{$i}}" style="display: flex; flex-direction: column; align-items: center; gap: 4px;">
              <span class="team-num" style="font-size: 0.85rem; font-weight: bold;"></span>
              <button class="scoring-button status-toggle" id="%s-{{add $i -1}}" onclick="toggleBoolStatus('%s', {{add $i -1}});" ontouchstart disabled style="width: 60px; height: 45px;"></button>
            </div>
            {{end}}
          </div>
        </div>
`, status.ID, status.DisplayName, status.ID, status.ID))
				}
			}
		}
		sb.WriteString("      </div>\n    </section>\n")
	}

	sb.WriteString(`  </div>
  <div id="panel-actions">
    <button id="commit" onclick="commitMatchScore();" ontouchstart disabled>Commit</button>
    <button id="fouls-button" class="scoring-button" onclick="showFoulsDialog();" ontouchstart disabled>Fouls</button>
  </div>
</main>

<dialog id="fouls-dialog" onclick="closeFoulsDialogIfOutside(event);">
  <div class="dialog-container">
    <div class="dialog-banner">Add Fouls</div>
    <div id="foul-container">
      {{template "foulButton" (dict "id" "foul-blue-minor" "color" "blue" "label" "Blue" "isMajor" false)}}
      {{template "foulButton" (dict "id" "foul-red-minor" "color" "red" "label" "Red" "isMajor" false)}}
      {{template "foulButton" (dict "id" "foul-blue-major" "color" "blue" "label" "Blue Major" "isMajor" true)}}
      {{template "foulButton" (dict "id" "foul-red-major" "color" "red" "label" "Red Major" "isMajor" true)}}
    </div>
    <button class="dialog-close" autofocus onclick="closeFoulsDialog();" ontouchstart>Close</button>
  </div>
</dialog>
{{end}}

{{define "script"}}
<script src="/static/js/match_timing.js"></script>
<script src="/static/js/scoring_panel.js"></script>
<script src="/static/js/generated_scoring_panel.js"></script>
{{end}}

{{define "foulButton"}}
<button id="{{.id}}" class="foul-button {{.color}} scoring-button" ontouchstart onclick="addFoul('{{.color}}', {{.isMajor}});">
  <div class="foul-button-label">{{.label}}</div>
  <div class="foul-button-counters">
    <span class="fouls-local">0</span> / <span class="fouls-global">0</span>
  </div>
</button>
{{end}}
`)

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

func generateScoringPanelJS(yamlData *GameYAML, staticJsDir string) error {
	filePath := filepath.Join(staticJsDir, "generated_scoring_panel.js")

	var sb strings.Builder
	sb.WriteString(`// Code generated by cmd/generate from game/game.yaml. DO NOT EDIT.
// Regenerate: go generate ./generate/

// Loading this file at all means custom game mode is active; flip the flag declared by the
// shared scoring_panel.js so it dispatches to the generated handler below explicitly.
IS_CUSTOM_GAME_MODE = true;

const boolStatuses = {};

const adjustCount = function (id, phase, delta) {
  websocket.send("adjustCount", {Id: id, Phase: phase, Delta: delta});
};

const toggleBoolStatus = function (id, robotIndex) {
  const key = id + "-" + robotIndex;
  const current = !!boolStatuses[key];
  websocket.send("setStatus", {Id: id, RobotIndex: robotIndex, Value: !current});
};

const setEnumStatus = function (id, robotIndex, valueId) {
  websocket.send("setEnumStatus", {Id: id, RobotIndex: robotIndex, ValueId: valueId});
};

const handleRealtimeScoreGenerated = function (data) {
  let score;
  if (alliance === "red") {
    score = data.Red.Score;
  } else {
    score = data.Blue.Score;
  }

  // Update counts
`)

	for _, sc := range yamlData.ScoringCounts {
		camel := toCamelCase(sc.ID)
		if sc.Phase == "auto" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("  $(`#%s-auto-count`).text(score.Auto%sCount);\n", sc.ID, camel))
		}
		if sc.Phase == "teleop" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("  $(`#%s-teleop-count`).text(score.Teleop%sCount);\n", sc.ID, camel))
		}
	}
	for _, ec := range yamlData.EndgameCounts {
		camel := toCamelCase(ec.ID)
		sb.WriteString(fmt.Sprintf("  $(`#%s-endgame-count`).text(score.Endgame%sCount);\n", ec.ID, camel))
	}

	sb.WriteString("\n  // Update statuses\n")
	for _, status := range yamlData.Statuses {
		camel := toCamelCase(status.ID)
		if len(status.Values) >= 2 {
			sb.WriteString("  for (let i = 0; i < 3; i++) {\n")
			for valIdx, val := range status.Values {
				sb.WriteString(fmt.Sprintf("    $(`#%s-${i}-${%q}`).attr('data-selected', score.%sStatuses[i] === %d);\n", status.ID, val.ID, camel, valIdx))
			}
			sb.WriteString("  }\n")
		} else {
			sb.WriteString(fmt.Sprintf("  for (let i = 0; i < 3; i++) {\n"+
				"    const val = !!score.%sStatuses[i];\n"+
				"    boolStatuses['%s-' + i] = val;\n"+
				"    $(\"#%s-\" + i).attr('data-selected', val);\n"+
				"  }\n", camel, status.ID, status.ID))
		}
	}

	sb.WriteString(`
  const redFouls = data.Red.Score.Fouls || [];
  const blueFouls = data.Blue.Score.Fouls || [];
  renderGlobalFoulCounts(redFouls, blueFouls);
};
`)

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

func generateAudienceDisplayTemplate(yamlData *GameYAML, templatesDir string) error {
	filePath := filepath.Join(templatesDir, "generated_audience_display.html")

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
  <head>
    <title>Audience Display - {{.EventSettings.Name}} - Cheesy Arena </title>
    <link rel="shortcut icon" href="/static/img/favicon.ico">
    <link rel="stylesheet" href="/static/css/lib/bootstrap.min.css"/>
    <link rel="stylesheet" href="/static/css/cheesy-arena.css"/>
    <link rel="stylesheet" href="/static/css/audience_display.css"/>
    <style>
      /* The custom game logo is a square/circular badge, so size it to fully cover its circle
         instead of the shared letterboxed-wide-logo sizing used by the default FRC logo.
         "top" must be !important: generated_audience_display.js's intro/match transitions
         (logoUp/logoDown) animate #logo's "top" via inline style to make room for the FRC
         banner logo's matchTime text below it. A circular logo has no such slack — object-fit:
         cover means any "top" shift just crops it — so pin it in place regardless. */
      #logo {
        top: 0 !important;
        width: 150px;
        height: 150px;
        object-fit: cover;
        border-radius: 50%;
      }
      #blindsLogo {
        top: 0;
        width: 310px;
        height: 310px;
        object-fit: cover;
        border-radius: 50%;
      }
    </style>
  </head>
  <body>
    <div id="overlayCentering">
      <div id="matchOverlayContainer">
        <div class="playoff-alliance" id="leftPlayoffAlliance"></div>
        <div id="matchOverlay">
          <div id="matchOverlayTop">
            <div class="teams" id="leftTeams">
              <div id="leftTeam1"></div>
              <div id="leftTeam2"></div>
              <div id="leftTeam3"></div>
            </div>
            <div class="score reversible-left">
              <div class="avatars">
                <img class="avatar" id="leftTeam1Avatar" src=""/>
                <img class="avatar" id="leftTeam2Avatar" src=""/>
                <img class="avatar" id="leftTeam3Avatar" src=""/>
              </div>
              <div class="score-fields" style="display: flex; gap: 15px; align-items: center;">
`)

	pieceGroups := buildPieceGroups(yamlData)

	for _, pg := range pieceGroups {
		camel := toCamelCase(pg.ID)
		sb.WriteString(fmt.Sprintf(`                <div class="live-counter-box" style="display: flex; flex-direction: column; align-items: center; justify-content: center; background: rgba(0,0,0,0.4); border: 2px solid rgba(255,255,255,0.2); border-radius: 4px; min-width: 70px; height: 75px; padding: 4px;">
                  <span style="font-size: 10px; color: #ccc; text-transform: uppercase; font-weight: bold; text-align: center;">%s</span>
                  <span id="left%sCount" style="font-size: 28px; font-weight: bold; color: white;">0</span>
                </div>
`, pg.DisplayName, camel))
	}

	sb.WriteString(`              </div>
              <div class="score-number" id="leftScoreNumber"></div>
            </div>
            <div class="score score-right reversible-right">
              <div class="score-number" id="rightScoreNumber"></div>
              <div class="score-fields" style="display: flex; gap: 15px; align-items: center;">
`)

	for _, pg := range pieceGroups {
		camel := toCamelCase(pg.ID)
		sb.WriteString(fmt.Sprintf(`                <div class="live-counter-box" style="display: flex; flex-direction: column; align-items: center; justify-content: center; background: rgba(0,0,0,0.4); border: 2px solid rgba(255,255,255,0.2); border-radius: 4px; min-width: 70px; height: 75px; padding: 4px;">
                  <span style="font-size: 10px; color: #ccc; text-transform: uppercase; font-weight: bold; text-align: center;">%s</span>
                  <span id="right%sCount" style="font-size: 28px; font-weight: bold; color: white;">0</span>
                </div>
`, pg.DisplayName, camel))
	}

	sb.WriteString(`              </div>
              <div class="avatars">
                <img class="avatar" id="rightTeam1Avatar" src=""/>
                <img class="avatar" id="rightTeam2Avatar" src=""/>
                <img class="avatar" id="rightTeam3Avatar" src=""/>
              </div>
            </div>
            <div class="teams" id="rightTeams">
              <div id="rightTeam1"></div>
              <div id="rightTeam2"></div>
              <div id="rightTeam3"></div>
            </div>
          </div>
          <div id="eventMatchInfo">
            <span>{{.EventSettings.Name}}</span>
            <span id="matchName"></span>
          </div>
        </div>
        <div class="playoff-alliance" id="rightPlayoffAlliance"></div>
      </div>
      <div id="playoffSeriesStatus">
        <span id="leftPlayoffAllianceWins"></span>&nbsp;-&nbsp;<span id="rightPlayoffAllianceWins"></span>
      </div>
      <div class="text-center" id="matchCircle">
        <img id="logo" src="/static/img/game-logo.png" alt="logo"/>
        <div id="matchTime"></div>
      </div>
      <div id="timeoutDetails">
        <div class="timeout-detail" id="timeoutBreakDescription"></div>
        <div class="timeout-detail" id="timeoutNextMatch">
          Next Up:<br/>
          <span id="timeoutNextMatchName"></span>
        </div>
      </div>
    </div>
    <div id="blindsContainer">
      <div class="blinds right background">
        <div class="blindsCenter blank"></div>
      </div>
      <div class="blinds left background">
        <div class="blindsCenter blank"></div>
      </div>
      <div class="blindsCenter full">
        <img id="blindsLogo" src="/static/img/blinds-logo.png" alt="logo"/>
      </div>
      <div id="finalScoreCentering">
        <div id="finalScore">
          <div id="finalResultIndicators">
            <div class="final-result-indicator-slot">
              <div class="final-result-indicator" id="leftFinalResultIndicator"></div>
            </div>
            <div class="final-result-indicator-slot">
              <div class="final-result-indicator" id="rightFinalResultIndicator"></div>
            </div>
          </div>
          <div class="final-score-row">
            <div class="final-score reversible-left" id="leftFinalScore"></div>
            <div class="final-score reversible-right" id="rightFinalScore"></div>
          </div>
          <div class="final-score-row">
            <div class="final-breakdown final-breakdown-teams">
              <div class="final-teams reversible-left">
                <div class="final-alliance playoff-only-field" id="leftFinalAlliance"></div>
                {{range $i := seq 4}}
                <div class="final-team-row">
                  <img class="final-team-avatar" id="leftFinalTeam{{$i}}Avatar" src=""/>
                  <div class="final-team-number" id="leftFinalTeam{{$i}}"></div>
                  <div class="final-team-card">
                    <div id="leftFinalTeam{{$i}}Card"></div>
                  </div>
                  <div class="final-team-rank playoff-hidden-field">
                    <img id="leftFinalTeam{{$i}}RankIndicator" src=""/>
                    <div id="leftFinalTeam{{$i}}RankNumber"></div>
                  </div>
                </div>
                {{end}}
              </div>
              <div class="playoff-only-field">
                <div class="final-destination" id="leftFinalDestination"></div>
              </div>
            </div>
`)

	// Left final breakdown values
	sb.WriteString("            <div class=\"final-breakdown\" id=\"leftFinalBreakdown\">\n")
	for _, pg := range pieceGroups {
		sb.WriteString(fmt.Sprintf("              <div id=\"leftFinal%sPoints\">0</div>\n", toCamelCase(pg.ID)))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("              <div id=\"leftFinal%sPoints\">0</div>\n", toCamelCase(ec.ID)))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("              <div id=\"leftFinal%sPoints\">0</div>\n", toCamelCase(status.ID)))
	}
	sb.WriteString("              <div id=\"leftFinalFoulPoints\">0</div>\n")
	sb.WriteString("              <div class=\"playoff-hidden-field\">\n")
	for _, rp := range yamlData.RPs {
		sb.WriteString(fmt.Sprintf("                <div id=\"leftFinal%sRankingPoint\">&#x2718;</div>\n", toCamelCase(rp.ID)))
	}
	sb.WriteString("                <div id=\"leftFinalRankingPoints\">0</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("              <div class=\"playoff-only-field\">\n")
	sb.WriteString("                <div>&nbsp;</div>\n")
	sb.WriteString("                <div id=\"leftFinalWins\">0</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("            </div>\n")

	// Center final breakdown labels
	sb.WriteString("            <div class=\"final-breakdown\" id=\"centerFinalBreakdown\" style=\"text-align: center;\">\n")
	for _, pg := range pieceGroups {
		sb.WriteString(fmt.Sprintf("              <div>%s</div>\n", pg.DisplayName))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("              <div>%s</div>\n", ec.DisplayName))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("              <div>%s</div>\n", status.DisplayName))
	}
	sb.WriteString("              <div>Foul</div>\n")
	sb.WriteString("              <div class=\"playoff-hidden-field\">\n")
	for _, rp := range yamlData.RPs {
		sb.WriteString(fmt.Sprintf("                <div>%s</div>\n", rp.DisplayName))
	}
	sb.WriteString("                <div>Ranking Points</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("              <div class=\"playoff-only-field\">\n")
	sb.WriteString("                <div>&nbsp;</div>\n")
	sb.WriteString("                <div>Wins</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("            </div>\n")

	// Right final breakdown values
	sb.WriteString("            <div class=\"final-breakdown\" id=\"rightFinalBreakdown\" style=\"text-align: right;\">\n")
	for _, pg := range pieceGroups {
		sb.WriteString(fmt.Sprintf("              <div id=\"rightFinal%sPoints\">0</div>\n", toCamelCase(pg.ID)))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("              <div id=\"rightFinal%sPoints\">0</div>\n", toCamelCase(ec.ID)))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("              <div id=\"rightFinal%sPoints\">0</div>\n", toCamelCase(status.ID)))
	}
	sb.WriteString("              <div id=\"rightFinalFoulPoints\">0</div>\n")
	sb.WriteString("              <div class=\"playoff-hidden-field\">\n")
	for _, rp := range yamlData.RPs {
		sb.WriteString(fmt.Sprintf("                <div id=\"rightFinal%sRankingPoint\">&#x2718;</div>\n", toCamelCase(rp.ID)))
	}
	sb.WriteString("                <div id=\"rightFinalRankingPoints\">0</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("              <div class=\"playoff-only-field\">\n")
	sb.WriteString("                <div>&nbsp;</div>\n")
	sb.WriteString("                <div id=\"rightFinalWins\">0</div>\n")
	sb.WriteString("              </div>\n")
	sb.WriteString("            </div>\n")

	sb.WriteString(`            <div class="final-breakdown final-breakdown-teams">
              <div class="final-teams reversible-right">
                <div class="final-alliance playoff-only-field" id="rightFinalAlliance"></div>
                {{range $i := seq 4}}
                <div class="final-team-row">
                  <img class="final-team-avatar" id="rightFinalTeam{{$i}}Avatar" src=""/>
                  <div class="final-team-number" id="rightFinalTeam{{$i}}"></div>
                  <div class="final-team-card">
                    <div id="rightFinalTeam{{$i}}Card"></div>
                  </div>
                  <div class="final-team-rank playoff-hidden-field">
                    <img id="rightFinalTeam{{$i}}RankIndicator" src=""/>
                    <div id="rightFinalTeam{{$i}}RankNumber"></div>
                  </div>
                </div>
                {{end}}
              </div>
              <div class="playoff-only-field">
                <div class="final-destination" id="rightFinalDestination"></div>
              </div>
            </div>
          </div>
          <div class="final-score-row" id="finalEventMatchInfo">
            <div class="final-footer">{{.EventSettings.Name}}</div>
            <div class="final-footer" id="finalMatchName">&nbsp;</div>
          </div>
          <div id="finalTiebreakReason"></div>
        </div>
      </div>
      <div id="bracket">
        <img id="bracketSvg" src=""/>
      </div>
      <div id="sponsor" class="carousel slide" data-bs-ride="carousel">
        <div class="carousel-inner" id="sponsorContainer">
        </div>
      </div>
    </div>
    <div id="allianceSelectionCentering" style="display: none;">
      <div id="allianceSelection"></div>
    </div>
    <div id="allianceRankingsCentering" {{if .SelectionShowUnpickedTeams}}class="enabled" {{end}}
      style="display: none;">
      <div id="allianceRankings"></div>
    </div>
    <div id="lowerThird">
      <img id="lowerThirdLogo" src="/static/img/lower-third-logo.png" alt="logo"/>
      <div id="lowerThirdTop"></div>
      <div id="lowerThirdBottom"></div>
      <div id="lowerThirdSingle"></div>
    </div>
    <script id="allianceSelectionTemplate" type="text/x-handlebars-template">
      <table id="allianceSelectionTable">
        {{"{{#each alliances}}"}}
        <tr>
          <td class="alliance-cell">{{"{{Index}}"}}</td>
          {{"{{#each this.TeamIds}}"}}
          <td class="selection-cell">{{"{{#if this}}"}}{{"{{this}}"}}{{"{{/if}}"}}</td>
          {{"{{/each}}"}}
        </tr>
        {{"{{/each}}"}}
        <tr>
          <td id="allianceSelectionTimer" colspan="{{"{{numColumns}}"}}"></td>
        </tr>
      </table>
    </script>
    <script id="sponsorImageTemplate" type="text/x-handlebars-template">
      <div class="carousel-item{{"{{#if First}}"}} active{{"{{/if}}"}}" data-bs-interval="{{"{{DisplayTimeMs}}"}}">
      <div class="sponsor-image-container">
        <img src="/static/img/sponsors/{{"{{Image}}"}}" />
      </div>
      <h1>{{"{{Subtitle}}"}}</h1>
      </div>
    </script>
    <script id="sponsorTextTemplate" type="text/x-handlebars-template">
      <div class="carousel-item{{"{{#if First}}"}} active{{"{{/if}}"}}" data-bs-interval="{{"{{DisplayTimeMs}}"}}">
      <h2>{{"{{Line1}}"}}<br/>{{"{{Line2}}"}}</h2>
      <h1>{{"{{Subtitle}}"}}</h1>
      </div>
    </script>
    {{range $sound := .MatchSounds}}
    <audio id="sound-{{$sound.Name}}" src="/static/audio/{{$sound.Name}}.{{$sound.FileExtension}}" preload="auto">
    </audio>
    {{end}}
    <script src="/static/js/lib/jquery.min.js"></script>
    <script src="/static/js/lib/jquery.json-2.4.min.js"></script>
    <script src="/static/js/lib/jquery.websocket-0.0.1.js"></script>
    <script src="/static/js/lib/jquery.transit.min.js"></script>
    <script src="/static/js/lib/handlebars-1.3.0.js"></script>
    <script src="/static/js/lib/bootstrap.bundle.min.js"></script>
    <script src="/static/js/cheesy-websocket.js"></script>
    <script src="/static/js/match_timing.js"></script>
    <script src="/static/js/generated_audience_display.js"></script>
  </body>
</html>
`)

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

func generateAudienceDisplayJS(yamlData *GameYAML, staticJsDir string) error {
	filePath := filepath.Join(staticJsDir, "generated_audience_display.js")

	stdJsPath := filepath.Join(staticJsDir, "audience_display.js")
	stdJsBytes, err := os.ReadFile(stdJsPath)
	if err != nil {
		return err
	}
	content := string(stdJsBytes)

	var rsSb strings.Builder
	rsSb.WriteString(`const handleRealtimeScore = function (data) {
  $("#leftScoreNumber").text(data.Red.ScoreSummary.Score);
  $("#rightScoreNumber").text(data.Blue.ScoreSummary.Score);

`)

	pieceGroups := buildPieceGroups(yamlData)

	for _, pg := range pieceGroups {
		camelGroup := toCamelCase(pg.ID)
		var redTerms []string
		var blueTerms []string
		for _, elID := range pg.ElementIDs {
			var el *Element
			for i := range yamlData.ScoringCounts {
				if yamlData.ScoringCounts[i].ID == elID {
					el = &yamlData.ScoringCounts[i]
				}
			}
			if el != nil {
				camelEl := toCamelCase(el.ID)
				if el.Phase == "auto" || el.Phase == "both" {
					redTerms = append(redTerms, fmt.Sprintf("(data.Red.Score.Auto%sCount || 0)", camelEl))
					blueTerms = append(blueTerms, fmt.Sprintf("(data.Blue.Score.Auto%sCount || 0)", camelEl))
				}
				if el.Phase == "teleop" || el.Phase == "both" {
					redTerms = append(redTerms, fmt.Sprintf("(data.Red.Score.Teleop%sCount || 0)", camelEl))
					blueTerms = append(blueTerms, fmt.Sprintf("(data.Blue.Score.Teleop%sCount || 0)", camelEl))
				}
			}
		}
		rsSb.WriteString(fmt.Sprintf("  $(`#left%sCount`).text(%s);\n", camelGroup, strings.Join(redTerms, " + ")))
		rsSb.WriteString(fmt.Sprintf("  $(`#right%sCount`).text(%s);\n", camelGroup, strings.Join(blueTerms, " + ")))
	}
	rsSb.WriteString("};")

	var spSb strings.Builder
	spSb.WriteString(`const handleScorePosted = function (data) {
  if (data.RedWon) {
    setFinalResultIndicator(redSide, "WINNER", "winner");
    setFinalResultIndicator(blueSide, "", "");
  } else if (data.BlueWon) {
    setFinalResultIndicator(redSide, "", "");
    setFinalResultIndicator(blueSide, "WINNER", "winner");
  } else {
    setFinalResultIndicator(redSide, "TIE", "tie");
    setFinalResultIndicator(blueSide, "TIE", "tie");
  }
  const tiebreakReason = data.TiebreakReason || "";
  $("#finalTiebreakReason").text(tiebreakReason);
  $("#finalTiebreakReason").attr("data-visible", tiebreakReason !== "");

  $("#leftFinalScore").text(data.RedScoreSummary.Score);
  $("#leftFinalAlliance").text("Alliance " + data.Match.PlayoffRedAlliance);
  setTeamInfo("left", 1, data.Match.Red1, data.RedCards, data.RedRankings);
  setTeamInfo("left", 2, data.Match.Red2, data.RedCards, data.RedRankings);
  setTeamInfo("left", 3, data.Match.Red3, data.RedCards, data.RedRankings);
  if (data.RedOffFieldTeamIds.length > 0) {
    setTeamInfo("left", 4, data.RedOffFieldTeamIds[0], data.RedCards, data.RedRankings);
  } else {
    setTeamInfo("left", 4, 0, data.RedCards, data.RedRankings);
  }

  $("#rightFinalScore").text(data.BlueScoreSummary.Score);
  $("#rightFinalAlliance").text("Alliance " + data.Match.PlayoffBlueAlliance);
  setTeamInfo("right", 1, data.Match.Blue1, data.BlueCards, data.BlueRankings);
  setTeamInfo("right", 2, data.Match.Blue2, data.BlueCards, data.BlueRankings);
  setTeamInfo("right", 3, data.Match.Blue3, data.BlueCards, data.BlueRankings);
  if (data.BlueOffFieldTeamIds.length > 0) {
    setTeamInfo("right", 4, data.BlueOffFieldTeamIds[0], data.BlueCards, data.BlueRankings);
  } else {
    setTeamInfo("right", 4, 0, data.BlueCards, data.BlueRankings);
  }

  // Populate Red (left) breakdown points
`)

	for _, pg := range pieceGroups {
		var terms []string
		for _, elID := range pg.ElementIDs {
			terms = append(terms, fmt.Sprintf("(data.RedScoreSummary.%sPoints || 0)", toCamelCase(elID)))
		}
		spSb.WriteString(fmt.Sprintf("  $(`#leftFinal%sPoints`).text(%s);\n", toCamelCase(pg.ID), strings.Join(terms, " + ")))
	}
	for _, ec := range yamlData.EndgameCounts {
		spSb.WriteString(fmt.Sprintf("  $(`#leftFinal%sPoints`).text(data.RedScoreSummary.%sPoints);\n", toCamelCase(ec.ID), toCamelCase(ec.ID)))
	}
	for _, status := range yamlData.Statuses {
		spSb.WriteString(fmt.Sprintf("  $(`#leftFinal%sPoints`).text(data.RedScoreSummary.%sPoints);\n", toCamelCase(status.ID), toCamelCase(status.ID)))
	}
	spSb.WriteString("  $(`#leftFinalFoulPoints`).text(data.RedScoreSummary.FoulPoints);\n")
	for _, rp := range yamlData.RPs {
		camelRP := toCamelCase(rp.ID) + "RankingPoint"
		spSb.WriteString(fmt.Sprintf("  $(`#leftFinal%sRankingPoint`).html(data.RedScoreSummary.%s ? \"&#x2714;\" : \"&#x2718;\");\n", toCamelCase(rp.ID), camelRP))
		spSb.WriteString(fmt.Sprintf("  $(`#leftFinal%sRankingPoint`).attr('data-checked', data.RedScoreSummary.%s);\n", toCamelCase(rp.ID), camelRP))
	}
	spSb.WriteString("  $(`#leftFinalRankingPoints`).html(data.RedRankingPoints);\n")
	spSb.WriteString("  $(`#leftFinalWins`).text(data.RedWins);\n")
	spSb.WriteString("  const redFinalDestination = $(`#leftFinalDestination`);\n")
	spSb.WriteString("  redFinalDestination.html(data.RedDestination.replace(\"Advances to \", \"Advances to<br>\"));\n")
	spSb.WriteString("  redFinalDestination.toggle(data.RedDestination !== \"\");\n")
	spSb.WriteString("  redFinalDestination.attr(\"data-won\", data.RedWon);\n\n")

	spSb.WriteString("  // Populate Blue (right) breakdown points\n")
	for _, pg := range pieceGroups {
		var terms []string
		for _, elID := range pg.ElementIDs {
			terms = append(terms, fmt.Sprintf("(data.BlueScoreSummary.%sPoints || 0)", toCamelCase(elID)))
		}
		spSb.WriteString(fmt.Sprintf("  $(`#rightFinal%sPoints`).text(%s);\n", toCamelCase(pg.ID), strings.Join(terms, " + ")))
	}
	for _, ec := range yamlData.EndgameCounts {
		spSb.WriteString(fmt.Sprintf("  $(`#rightFinal%sPoints`).text(data.BlueScoreSummary.%sPoints);\n", toCamelCase(ec.ID), toCamelCase(ec.ID)))
	}
	for _, status := range yamlData.Statuses {
		spSb.WriteString(fmt.Sprintf("  $(`#rightFinal%sPoints`).text(data.BlueScoreSummary.%sPoints);\n", toCamelCase(status.ID), toCamelCase(status.ID)))
	}
	spSb.WriteString("  $(`#rightFinalFoulPoints`).text(data.BlueScoreSummary.FoulPoints);\n")
	for _, rp := range yamlData.RPs {
		camelRP := toCamelCase(rp.ID) + "RankingPoint"
		spSb.WriteString(fmt.Sprintf("  $(`#rightFinal%sRankingPoint`).html(data.BlueScoreSummary.%s ? \"&#x2714;\" : \"&#x2718;\");\n", toCamelCase(rp.ID), camelRP))
		spSb.WriteString(fmt.Sprintf("  $(`#rightFinal%sRankingPoint`).attr('data-checked', data.BlueScoreSummary.%s);\n", toCamelCase(rp.ID), camelRP))
	}
	spSb.WriteString("  $(`#rightFinalRankingPoints`).html(data.BlueRankingPoints);\n")
	spSb.WriteString("  $(`#rightFinalWins`).text(data.BlueWins);\n")
	spSb.WriteString("  const blueFinalDestination = $(`#rightFinalDestination`);\n")
	spSb.WriteString("  blueFinalDestination.html(data.BlueDestination.replace(\"Advances to \", \"Advances to<br>\"));\n")
	spSb.WriteString("  blueFinalDestination.toggle(data.BlueDestination !== \"\");\n")
	spSb.WriteString("  blueFinalDestination.attr(\"data-won\", data.BlueWon);\n\n")

	spSb.WriteString(`  let matchName = data.Match.LongName;
  if (data.Match.NameDetail !== "") {
    matchName += " &ndash; " + data.Match.NameDetail;
  }
  $("#finalMatchName").html(matchName);

  // Reload the bracket to reflect any changes.
  $("#bracketSvg").attr("src", "/api/bracket/svg?activeMatch=saved&v=" + new Date().getTime());

  if (data.Match.Type === matchTypePlayoff) {
    $(".playoff-hidden-field").hide();
    $(".playoff-only-field").show();
  } else {
    $(".playoff-hidden-field").show();
    $(".playoff-only-field").hide();
  }
};`)

	reRealtimeScore := regexp.MustCompile(`const handleRealtimeScore = function\s*\(data\)\s*\{[\s\S]*?\};\n`)
	content = reRealtimeScore.ReplaceAllString(content, rsSb.String()+"\n")

	reScorePosted := regexp.MustCompile(`const handleScorePosted = function\s*\(data\)\s*\{[\s\S]*?\n\};`)
	content = reScorePosted.ReplaceAllString(content, spSb.String())

	return os.WriteFile(filePath, []byte(content), 0644)
}

// generateRefereePanelTemplate emits templates/generated_referee_panel.html. The cards/fouls/
// control-button/modal markup is copied verbatim from templates/referee_panel.html (it has no
// game-specific content at all); only the scoreSummary block is generated per game.yaml, with one
// row per status (statuses are the only per-robot-indexed score data available to generate from).
func generateRefereePanelTemplate(yamlData *GameYAML, templatesDir string) error {
	filePath := filepath.Join(templatesDir, "generated_referee_panel.html")

	var sb strings.Builder
	sb.WriteString(`{{define "title"}}Custom Referee Panel{{end}}
{{define "body"}}
<div id="matchName"></div>
<div id="refereePanel">
  <div id="cards" class="headRef-dependent">
    <h3 id="teamTitle">Red/Yellow Cards</h3>
    <div class="alliance-cards" id="redCards">
      {{range $i := seq 3}}
      {{template "teamCard" dict "alliance" "red" "position" $i}}
      {{end}}
    </div>
    <div class="alliance-cards" id="blueCards">
      {{range $i := seq 3}}
      {{template "teamCard" dict "alliance" "blue" "position" $i}}
      {{end}}
    </div>
    <div id="scoringStatuses">
      <div class="scoring-status" id="redScoreStatus"></div>
      <div class="scoring-status" id="blueScoreStatus"></div>
    </div>
  </div>
  <div id="fouls">
    <div id="scoreSummary" class="headRef-dependent">
      {{template "scoreSummary" dict "id" "blueScoreSummary"}}
      {{template "scoreSummary" dict "id" "redScoreSummary"}}
    </div>
    <h3>Fouls</h3>
    <div id="foulButtons">
      <div class="foul-button blue-foul" onclick="addFoul('blue', false);">Blue</div>
      <div class="foul-button blue-foul" onclick="addFoul('blue', true);">Blue Major</div>
      <div class="foul-button red-foul" onclick="addFoul('red', false);">Red</div>
      <div class="foul-button red-foul" onclick="addFoul('red', true);">Red Major</div>
    </div>
    <div id="foulList"></div>
  </div>
</div>
<p>Note: Team and rule assignment are optional.</p>
<div id="controlButtons" class="headRef-dependent">
  <div class="control-button" id="volunteerButton" onclick="signalVolunteers();">Signal Count</div>
  <div class="control-button" id="resetButton" onclick="signalReset();">Signal Reset</div>
  <div class="control-button" id="commitButton" onclick="confirmCommit();">Commit & Post</div>
</div>
<div id="confirmCommit" class="modal">
  <div class="modal-dialog">
    <div class="modal-content">
      <div class="modal-header">
        <h4 class="modal-title">Scores not committed</h4>
        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
      </div>
      <div class="modal-body">
        Are you sure you want to commit and post without all scoring tablets having committed?
      </div>
      <div class="modal-footer">
        <form class="form-horizontal">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-primary ms-2" onclick="commitAndPost();" data-bs-dismiss="modal">
            Commit & Post
          </button>
        </form>
      </div>
    </div>
  </div>
</div>
<div id="confirmBypass" class="modal">
  <div class="modal-dialog">
    <div class="modal-content">
      <div class="modal-header">
        <h4 class="modal-title" id="confirmBypassTitle">Disable?</h4>
        <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
      </div>
      <div class="modal-footer">
        <form class="form-horizontal">
          <button type="button" class="btn btn-lg btn-secondary" data-bs-dismiss="modal">Cancel</button>
          <button type="button" class="btn btn-lg btn-danger ms-2" onclick="toggleBypass();" data-bs-dismiss="modal" id="confirmBypassAction">
            Disable
          </button>
        </form>
      </div>
    </div>
  </div>
</div>
{{end}}
{{define "head"}}
<link rel="manifest" href="/static/manifest/referee.manifest">
<meta name="viewport" content="width=device-width, user-scalable=no">
<link href="/static/css/referee_panel.css" rel="stylesheet">
{{end}}
{{define "script"}}
<script src="/static/js/match_timing.js"></script>
<script src="/static/js/referee_panel.js"></script>
<script src="/static/js/generated_referee_panel.js"></script>
{{end}}
{{define "teamCard"}}
<div class="team-card" id="{{.alliance}}{{.position}}Card" data-alliance="{{.alliance}}" data-station="{{slice .alliance 0 1}}{{.position}}" onclick="cycleCard(this);">
</div>
{{end}}
{{define "scoreSummary"}}
<div id="{{.id}}" class="scoreSummary">
  <div class="placeholder"></div>
  <div class="team-1">0</div>
  <div class="team-2">0</div>
  <div class="team-3">0</div>
`)

	// Alliance-level scoring totals — these aren't robot-indexed, so they render as a single
	// wide-row value rather than three per-robot columns. Grouped by phase and reported as raw
	// counts per individual scoring element (not summed/grouped by game piece): a referee is
	// verifying discrete field actions, so e.g. mayhem-fms-2025's real referee panel shows Hull
	// and Deck counts side by side rather than a combined point total.
	for _, phase := range []struct{ id, label string }{{"auto", "Auto"}, {"teleop", "Teleop"}} {
		var elementIDs []string
		for _, sc := range yamlData.ScoringCounts {
			if sc.Phase == phase.id || sc.Phase == "both" {
				elementIDs = append(elementIDs, sc.ID)
			}
		}
		if len(elementIDs) == 0 {
			continue
		}
		placeholders := make([]string, len(elementIDs))
		for i := range placeholders {
			placeholders[i] = "0"
		}
		sb.WriteString(fmt.Sprintf(
			"\n  <div class=\"label\">%s</div>\n  <div class=\"wide-row count-total phase-%s\">%s</div>\n",
			phase.label, phase.id, strings.Join(placeholders, " / "),
		))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf(
			"\n  <div class=\"label\">%s</div>\n  <div class=\"wide-row count-total endgame-%s\">0</div>\n",
			ec.DisplayName, ec.ID,
		))
	}

	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("\n  <div class=\"label\">%s</div>\n", status.DisplayName))
		for i := 1; i <= 3; i++ {
			sb.WriteString(fmt.Sprintf(
				"  <div class=\"status-badge team-%d-%s\" data-active=\"false\">-</div>\n", i, status.ID,
			))
		}
	}

	sb.WriteString(`</div>
{{end}}
`)

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

// generateRefereePanelJS emits static/js/generated_referee_panel.js, which defines
// updateScoreSummaryGenerated(scoreRoot, score). The shared static/js/referee_panel.js dispatches
// to this function instead of its own FRC-specific tower-status logic when IS_CUSTOM_GAME_MODE is
// true (same explicit shared/generated dispatch pattern used by the scoring panel).
func generateRefereePanelJS(yamlData *GameYAML, staticJsDir string) error {
	filePath := filepath.Join(staticJsDir, "generated_referee_panel.js")

	var sb strings.Builder
	sb.WriteString(`// Code generated by cmd/generate from game/game.yaml. DO NOT EDIT.
// Regenerate: go generate ./generate/

// Loading this file at all means custom game mode is active; flip the flag declared by the
// shared referee_panel.js so it dispatches to the generated handler below explicitly.
IS_CUSTOM_GAME_MODE = true;

const updateScoreSummaryGenerated = function (scoreRoot, score) {
`)

	for _, phase := range []struct{ id, fieldPrefix string }{{"auto", "Auto"}, {"teleop", "Teleop"}} {
		var terms []string
		for _, sc := range yamlData.ScoringCounts {
			if sc.Phase == phase.id || sc.Phase == "both" {
				terms = append(terms, fmt.Sprintf("score.%s%sCount", phase.fieldPrefix, toCamelCase(sc.ID)))
			}
		}
		if len(terms) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf(
			"  $(`#${scoreRoot} .phase-%s`).text([%s].join(\" / \"));\n", phase.id, strings.Join(terms, ", "),
		))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf(
			"  $(`#${scoreRoot} .endgame-%s`).text(score.Endgame%sCount);\n", ec.ID, toCamelCase(ec.ID),
		))
	}

	for _, status := range yamlData.Statuses {
		camel := toCamelCase(status.ID)
		if len(status.Values) >= 2 {
			names := make([]string, len(status.Values))
			for i, val := range status.Values {
				names[i] = fmt.Sprintf("%q", val.DisplayName)
			}
			sb.WriteString(fmt.Sprintf("  const %sNames = [%s];\n", camel, strings.Join(names, ", ")))
			for i := 0; i < 3; i++ {
				sb.WriteString(fmt.Sprintf(
					"  $(`#${scoreRoot} .team-%d-%s`).text(%sNames[score.%sStatuses[%d]]);\n", i+1, status.ID, camel, camel, i,
				))
				sb.WriteString(fmt.Sprintf(
					"  $(`#${scoreRoot} .team-%d-%s`).attr('data-active', score.%sStatuses[%d] !== 0);\n", i+1, status.ID, camel, i,
				))
			}
		} else {
			for i := 0; i < 3; i++ {
				sb.WriteString(fmt.Sprintf(
					"  $(`#${scoreRoot} .team-%d-%s`).text(score.%sStatuses[%d] ? \"\\u2713\" : \"\\u2717\");\n",
					i+1, status.ID, camel, i,
				))
				sb.WriteString(fmt.Sprintf(
					"  $(`#${scoreRoot} .team-%d-%s`).attr('data-active', !!score.%sStatuses[%d]);\n", i+1, status.ID, camel, i,
				))
			}
		}
	}

	sb.WriteString("};\n")

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}

func generateTemplateTest(yamlData *GameYAML, destDir string) error {
	filePath := filepath.Join(destDir, "generated_template_test.go")

	var sb strings.Builder
	sb.WriteString(`// Code generated by cmd/generate from game/game.yaml. DO NOT EDIT.
// Regenerate: go generate ./generate/

package main

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratedTemplates_ParseAndContent(t *testing.T) {
	templatesDir := "../../templates"
	panelPath := filepath.Join(templatesDir, "generated_scoring_panel.html")
	audiencePath := filepath.Join(templatesDir, "generated_audience_display.html")

	{
		content, err := os.ReadFile(panelPath)
		assert.NoError(t, err)
		tmpl := template.New("scoring")
		tmpl.Funcs(template.FuncMap{
			"dict": func(values ...any) (map[string]any, error) { return nil, nil },
			"seq": func(n int) []int { return []int{0, 1, 2} },
			"add": func(a, b int) int { return a + b },
		})
		_, err = tmpl.Parse(string(content))
		assert.NoError(t, err)

`)

	for _, sc := range yamlData.ScoringCounts {
		if sc.Phase == "auto" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`onclick="adjustCount('%s', 'auto', -1);"`, sc.ID)))
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="%s-auto-count"`, sc.ID)))
		}
		if sc.Phase == "teleop" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`onclick="adjustCount('%s', 'teleop', -1);"`, sc.ID)))
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="%s-teleop-count"`, sc.ID)))
		}
	}
	for _, status := range yamlData.Statuses {
		if status.Phase == "auto" {
			if len(status.Values) >= 2 {
				for _, val := range status.Values {
					sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`onclick="setEnumStatus('%s', {{add $i -1}}, '%s');"`, status.ID, val.ID)))
				}
			} else {
				sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`onclick="toggleBoolStatus('%s', {{add $i -1}});"`, status.ID)))
			}
		}
	}

	sb.WriteString(`	}

	{
		content, err := os.ReadFile(audiencePath)
		assert.NoError(t, err)
		tmpl := template.New("audience")
		tmpl.Funcs(template.FuncMap{
			"seq": func(n int) []int { return []int{0, 1, 2, 3} },
			"add": func(a, b int) int { return a + b },
		})
		_, err = tmpl.Parse(string(content))
		assert.NoError(t, err)

`)

	pieceGroups := buildPieceGroups(yamlData)

	for _, pg := range pieceGroups {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="leftFinal%sPoints"`, toCamelCase(pg.ID))))
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="rightFinal%sPoints"`, toCamelCase(pg.ID))))
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="leftFinal%sPoints"`, toCamelCase(ec.ID))))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="leftFinal%sPoints"`, toCamelCase(status.ID))))
	}
	for _, rp := range yamlData.RPs {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`id="leftFinal%sRankingPoint"`, toCamelCase(rp.ID))))
	}

	sb.WriteString(`	}

	{
		refereePath := filepath.Join(templatesDir, "generated_referee_panel.html")
		content, err := os.ReadFile(refereePath)
		assert.NoError(t, err)
		tmpl := template.New("referee")
		tmpl.Funcs(template.FuncMap{
			"dict": func(values ...any) (map[string]any, error) { return nil, nil },
			"seq":  func(n int) []int { return []int{0, 1, 2} },
		})
		_, err = tmpl.Parse(string(content))
		assert.NoError(t, err)

		assert.Contains(t, string(content), "generated_referee_panel.js")
`)

	for _, phase := range []string{"auto", "teleop"} {
		hasPhase := false
		for _, sc := range yamlData.ScoringCounts {
			if sc.Phase == phase || sc.Phase == "both" {
				hasPhase = true
			}
		}
		if hasPhase {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`phase-%s`, phase)))
		}
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`endgame-%s`, ec.ID)))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`team-1-%s`, status.ID)))
	}

	sb.WriteString(`	}
}

func TestGeneratedJS_Content(t *testing.T) {
	jsDir := "../../static/js"
	panelJsPath := filepath.Join(jsDir, "generated_scoring_panel.js")
	audienceJsPath := filepath.Join(jsDir, "generated_audience_display.js")

	{
		content, err := os.ReadFile(panelJsPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "adjustCount = function")
		assert.Contains(t, string(content), "toggleBoolStatus = function")
		assert.Contains(t, string(content), "setEnumStatus = function")
`)

	for _, sc := range yamlData.ScoringCounts {
		camel := toCamelCase(sc.ID)
		if sc.Phase == "auto" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`score.Auto%sCount`, camel)))
		}
		if sc.Phase == "teleop" || sc.Phase == "both" {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`score.Teleop%sCount`, camel)))
		}
	}

	sb.WriteString(`	}

	{
		content, err := os.ReadFile(audienceJsPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "handleRealtimeScore = function")
		assert.Contains(t, string(content), "handleScorePosted = function")
`)

	for _, pg := range pieceGroups {
		camelGroup := toCamelCase(pg.ID)
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`left%sCount`, camelGroup)))
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`leftFinal%sPoints`, camelGroup)))
	}

	sb.WriteString(`	}

	{
		refereeJsPath := filepath.Join(jsDir, "generated_referee_panel.js")
		content, err := os.ReadFile(refereeJsPath)
		assert.NoError(t, err)
		assert.Contains(t, string(content), "updateScoreSummaryGenerated = function")
`)

	for _, phase := range []struct{ id, fieldPrefix string }{{"auto", "Auto"}, {"teleop", "Teleop"}} {
		hasPhase := false
		for _, sc := range yamlData.ScoringCounts {
			if sc.Phase == phase.id || sc.Phase == "both" {
				hasPhase = true
			}
		}
		if hasPhase {
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`phase-%s`, phase.id)))
			sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`score.%s`, phase.fieldPrefix)))
		}
	}
	for _, ec := range yamlData.EndgameCounts {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`endgame-%s`, ec.ID)))
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`score.Endgame%sCount`, toCamelCase(ec.ID))))
	}
	for _, status := range yamlData.Statuses {
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`team-1-%s`, status.ID)))
		sb.WriteString(fmt.Sprintf("\t\tassert.Contains(t, string(content), %q)\n", fmt.Sprintf(`score.%sStatuses`, toCamelCase(status.ID))))
	}

	sb.WriteString(`	}
}
`)

	return os.WriteFile(filePath, []byte(sb.String()), 0644)
}
