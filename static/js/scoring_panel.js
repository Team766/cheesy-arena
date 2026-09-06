// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
// Author: ian@yann.io (Ian Thompson)
//
// Client-side logic for the scoring interface.

var websocket;
let alliance;
let committed = false;

// True when the server is running a YAML-configured custom game; set from GET /api/game_config
// during page load, before the websocket is opened, so the realtime handler knows which score
// shape to expect.
let IS_CUSTOM_GAME_MODE = false;
// Custom game mode only: the near/far scorer hint this panel filters on ("" = show everything).
let panelScorer = "";
// The parsed /api/game_config document, used to build the panel and to iterate scoring elements.
window.gameConfig = null;
// Custom game mode only: mirror of the server's boolean statuses, keyed "<statusId>-<robotIndex>",
// so that a toggle press can send the opposite of the current value.
window.boolStatuses = {};

// True when scoring controls in general should be available
let scoringAvailable = false;
// True when the commit button should be available
let commitAvailable = false;

let localFoulCounts = {
  "red-minor": 0,
  "blue-minor": 0,
  "red-major": 0,
  "blue-major": 0,
}

const foulsDialog = $("#fouls-dialog")[0];
const showFoulsDialog = function () {
  foulsDialog.showModal();
}
const closeFoulsDialog = function () {
  foulsDialog.close();
}
const closeFoulsDialogIfOutside = function (event) {
  if (event.target === foulsDialog) {
    closeFoulsDialog();
  }
}

// Websocket message senders for the custom game mode controls.
const adjustCount = function (id, phase, delta) {
  websocket.send("adjustCount", {Id: id, Phase: phase, Delta: delta});
}

const toggleBoolStatus = function (id, robotIndex) {
  const key = id + "-" + robotIndex;
  const current = !!window.boolStatuses[key];
  websocket.send("setStatus", {Id: id, RobotIndex: robotIndex, Value: !current});
}

const setEnumStatus = function (id, robotIndex, valueId) {
  websocket.send("setEnumStatus", {Id: id, RobotIndex: robotIndex, ValueId: valueId});
}

const cycleEnumStatus = function (id, robotIndex) {
  websocket.send("cycleEnumStatus", {Id: id, RobotIndex: robotIndex});
}

// Handles a websocket message to update the teams for the current match.
const handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);
  if (alliance === "red") {
    $(".team-1 .team-num").text(data.Match.Red1);
    $(".team-2 .team-num").text(data.Match.Red2);
    $(".team-3 .team-num").text(data.Match.Red3);
  } else {
    $(".team-1 .team-num").text(data.Match.Blue1);
    $(".team-2 .team-num").text(data.Match.Blue2);
    $(".team-3 .team-num").text(data.Match.Blue3);
  }
};

const renderLocalFoulCounts = function () {
  for (const foulType in localFoulCounts) {
    const count = localFoulCounts[foulType];
    $(`#foul-${foulType} .fouls-local`).text(count)
  }
}

const renderGlobalFoulCounts = function (redFouls, blueFouls) {
  $(`#foul-blue-minor .fouls-global`).text(blueFouls.filter(foul => !foul.IsMajor).length)
  $(`#foul-blue-major .fouls-global`).text(blueFouls.filter(foul => foul.IsMajor).length)
  $(`#foul-red-minor .fouls-global`).text(redFouls.filter(foul => !foul.IsMajor).length)
  $(`#foul-red-major .fouls-global`).text(redFouls.filter(foul => foul.IsMajor).length)
}

const resetFoulCounts = function () {
  localFoulCounts["red-minor"] = 0;
  localFoulCounts["blue-minor"] = 0;
  localFoulCounts["red-major"] = 0;
  localFoulCounts["blue-major"] = 0;
  renderLocalFoulCounts();
}

const addFoul = function (alliance, isMajor) {
  const foulType = `${alliance}-${isMajor ? "major" : "minor"}`;
  localFoulCounts[foulType] += 1;
  renderLocalFoulCounts();
  websocket.send("addFoul", {Alliance: alliance, IsMajor: isMajor});
}

// Handles a websocket message to update the match status.
const handleMatchTime = function (data) {
  switch (matchStates[data.MatchState]) {
    case "AUTO_PERIOD":
    case "PAUSE_PERIOD":
    case "TELEOP_PERIOD":
      scoringAvailable = true;
      commitAvailable = false;
      committed = false;
      break;
    case "POST_MATCH":
      if (!committed) {
        scoringAvailable = true;
        commitAvailable = true;
      }
      break;
    default:
      scoringAvailable = false;
      commitAvailable = false;
      committed = false;
      resetFoulCounts();
  }
  updateUIMode();
};

// Clear any local ephemeral state that is not maintained by the server
const resetLocalState = function () {
  committed = false;
  updateUIMode();
}

// Refresh which UI controls are enabled/disabled
const updateUIMode = function () {
  $(".scoring-button").prop('disabled', !scoringAvailable);
  $(".scoring-tower-button").prop('disabled', !scoringAvailable);
  $("#commit").prop('disabled', !commitAvailable);
}

const endgameStatusNames = [
  "None",
  "Level 1",
  "Level 2",
  "Level 3",
];

// Handles a websocket message to update the realtime scoring fields.
const handleRealtimeScore = function (data) {
  let realtimeScore;
  if (alliance === "red") {
    realtimeScore = data.Red;
  } else {
    realtimeScore = data.Blue;
  }
  const score = realtimeScore.Score;

  for (let i = 0; i < 3; i++) {
    const i1 = i + 1;
    for (let j = 0; j < endgameStatusNames.length; j++) {
      $(`#auto-input-${i1} .tower-${j}`).attr("data-selected", j == score.AutoTowerStatuses[i]);
      $(`#endgame-input-${i1} .tower-${j}`).attr("data-selected", j == score.EndgameTowerStatuses[i]);
    }
  }

  const redFouls = data.Red.Score.Fouls || [];
  const blueFouls = data.Blue.Score.Fouls || [];
  renderGlobalFoulCounts(redFouls, blueFouls);
};

// Custom game mode equivalent of handleRealtimeScore. The custom game.Score serializes as
// Counts ("<countId>_<phase>" -> int), BoolStatuses (id -> [3]bool) and EnumStatuses
// (id -> [3]int, indexes into the status' configured values).
const handleRealtimeScoreCustom = function (data) {
  const realtimeScore = alliance === "red" ? data.Red : data.Blue;
  const score = realtimeScore.Score || {};
  const counts = score.Counts || {};
  const boolStatusesMap = score.BoolStatuses || {};
  const enumStatusesMap = score.EnumStatuses || {};

  if (window.gameConfig) {
    (window.gameConfig.scoring_counts || []).forEach(sc => {
      (sc.phases || []).forEach(pp => {
        const key = `${sc.id}_${pp.phase}`;
        const val = counts[key] !== undefined ? counts[key] : 0;
        $(`#${sc.id}-${pp.phase}-count`).text(val);
      });
    });

    (window.gameConfig.statuses || []).forEach(st => {
      if (st.values && st.values.length > 0) {
        const arr = enumStatusesMap[st.id] || [0, 0, 0];
        for (let i = 0; i < 3; i++) {
          const valIdx = arr[i] || 0;
          const valName = st.values[valIdx] ? st.values[valIdx].display_name : "None";
          $(`#${st.id}-${i}`).text(valName).attr("data-selected", valIdx !== 0);
        }
      } else {
        const arr = boolStatusesMap[st.id] || [false, false, false];
        for (let i = 0; i < 3; i++) {
          const val = !!arr[i];
          window.boolStatuses[`${st.id}-${i}`] = val;
          $(`#${st.id}-${i}`).attr("data-selected", val);
        }
      }
    });
  }

  const redFouls = data.Red.Score.Fouls || [];
  const blueFouls = data.Blue.Score.Fouls || [];
  renderGlobalFoulCounts(redFouls, blueFouls);
};

// Websocket message senders for various buttons
const handleAutoTowerClick = function (teamPosition, autoTowerStatus) {
  websocket.send("autoTower", {TeamPosition: teamPosition, AutoTowerStatus: autoTowerStatus});
}
const handleEndgameClick = function (teamPosition, endgameTowerStatus) {
  websocket.send("endgame", {TeamPosition: teamPosition, EndgameTowerStatus: endgameTowerStatus});
}

// Sends a websocket message to indicate that the score for this alliance is ready.
const commitMatchScore = function () {
  websocket.send("commitMatch");

  committed = true;
  scoringAvailable = false;
  commitAvailable = false;
  updateUIMode();
};

// Builds the scoring controls for a custom game from its configuration: one section per phase,
// containing a +/- control for every scoring count in that phase and a per-robot button row for
// every status in that phase. Presentation lives in static/css/custom_scoring_panel.css.
const buildCustomUI = function (config) {
  const container = $("#tower-controls");
  if (!container.length) return;
  container.empty();

  const phases = [
    {key: "auto", title: "Autonomous"},
    {key: "teleop", title: "Teleoperated"},
    {key: "endgame", title: "Endgame"},
  ];

  phases.forEach(phase => {
    // An element with a scorer hint only appears on the matching near/far panel; a panel with no
    // hint of its own (the plain red/blue panels) shows everything.
    const scoredHere = el => !panelScorer || !el.scorer || el.scorer === panelScorer;
    const phaseCounts = (config.scoring_counts || []).filter(sc =>
      scoredHere(sc) && (sc.phases || []).some(pp => pp.phase === phase.key)
    );
    const phaseStatuses = (config.statuses || []).filter(st =>
      scoredHere(st) && (st.phases || []).some(pp => pp.phase === phase.key)
    );

    if (phaseCounts.length === 0 && phaseStatuses.length === 0) return;

    let html = `<section class="tower-section" id="${phase.key}-section">`;
    html += `<h1>${phase.title}</h1>`;
    html += `<div class="custom-controls">`;

    phaseCounts.forEach(sc => {
      html += `<div class="count-control" id="${sc.id}-${phase.key}">`;
      html += `<span class="count-label">${sc.display_name}</span>`;
      html += `<div class="count-buttons">`;
      html += `<button class="scoring-button count-button" onclick="adjustCount('${sc.id}', '${phase.key}', -1);"` +
        ` ontouchstart disabled>-</button>`;
      html += `<span class="count-value" id="${sc.id}-${phase.key}-count">0</span>`;
      html += `<button class="scoring-button count-button" onclick="adjustCount('${sc.id}', '${phase.key}', 1);"` +
        ` ontouchstart disabled>+</button>`;
      html += `</div></div>`;
    });

    phaseStatuses.forEach(st => {
      html += `<div class="status-control" id="${st.id}-status">`;
      html += `<span class="status-label">${st.display_name}</span>`;
      html += `<div class="status-teams">`;

      for (let i = 0; i < 3; i++) {
        // The team-N class is what handleMatchLoad targets to fill in the team numbers.
        html += `<div class="status-team team-${i + 1}">`;
        html += `<span class="team-num"></span>`;
        if (st.values && st.values.length > 0) {
          const defaultLabel = st.values[0] ? st.values[0].display_name : "None";
          html += `<button class="scoring-button status-toggle enum-toggle" id="${st.id}-${i}"` +
            ` onclick="cycleEnumStatus('${st.id}', ${i});" ontouchstart disabled>${defaultLabel}</button>`;
        } else {
          html += `<button class="scoring-button status-toggle" id="${st.id}-${i}"` +
            ` onclick="toggleBoolStatus('${st.id}', ${i});" ontouchstart disabled></button>`;
        }
        html += `</div>`;
      }

      html += `</div></div>`;
    });

    html += `</div></section>`;
    container.append(html);
  });
  updateUIMode();
};

$(function () {
  position = window.location.href.split("/").slice(-1)[0];
  // Custom-game positions are "<alliance>_<near|far>"; the template says which alliance the panel
  // belongs to and which scorer hint it filters on. The stock template has neither attribute.
  alliance = $("main").data("alliance") || position;
  panelScorer = $("main").data("scorer") || "";
  $(".container").attr("data-alliance", alliance);
  resetLocalState();

  // Fetch the game configuration first so that the custom panel exists before the first realtime
  // score message arrives; the endpoint 404s in the stock build, in which case .done() is skipped.
  $.getJSON("/api/game_config")
    .done(function (config) {
      if (config && config.game && config.game.name) {
        IS_CUSTOM_GAME_MODE = true;
        window.gameConfig = config;
        buildCustomUI(config);
      }
    })
    .always(function () {
      // Set up the websocket back to the server.
      websocket = new CheesyWebsocket("/panels/scoring/" + position + "/websocket", {
        matchLoad: function (event) {
          handleMatchLoad(event.data);
        },
        matchTime: function (event) {
          handleMatchTime(event.data);
        },
        realtimeScore: function (event) {
          if (IS_CUSTOM_GAME_MODE) {
            handleRealtimeScoreCustom(event.data);
          } else {
            handleRealtimeScore(event.data);
          }
        },
        resetLocalState: function (event) {
          resetLocalState();
        },
      });
    });
});
