// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
// Author: ian@yann.io (Ian Thompson)
//
// Client-side logic for the scoring interface.

var websocket;
let alliance;
let committed = false;

let IS_CUSTOM_GAME_MODE = false;
window.boolStatuses = {};
window.gameConfig = null;

let scoringAvailable = false;
let commitAvailable = false;

let localFoulCounts = {
  "red-minor": 0,
  "blue-minor": 0,
  "red-major": 0,
  "blue-major": 0,
};

const foulsDialog = $("#fouls-dialog")[0];
const showFoulsDialog = function () {
  if (foulsDialog) foulsDialog.showModal();
};
const closeFoulsDialog = function () {
  if (foulsDialog) foulsDialog.close();
};
const closeFoulsDialogIfOutside = function (event) {
  if (event.target === foulsDialog) {
    closeFoulsDialog();
  }
};

const adjustCount = function (id, phase, delta) {
  websocket.send("adjustCount", {Id: id, Phase: phase, Delta: delta});
};

const toggleBoolStatus = function (id, robotIndex) {
  const key = id + "-" + robotIndex;
  const current = !!window.boolStatuses[key];
  websocket.send("setStatus", {Id: id, RobotIndex: robotIndex, Value: !current});
};

const setEnumStatus = function (id, robotIndex, valueId) {
  websocket.send("setEnumStatus", {Id: id, RobotIndex: robotIndex, ValueId: valueId});
};

const cycleEnumStatus = function (id, robotIndex) {
  websocket.send("cycleEnumStatus", {Id: id, RobotIndex: robotIndex});
};

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
    $(`#foul-${foulType} .fouls-local`).text(count);
  }
};

const renderGlobalFoulCounts = function (redFouls, blueFouls) {
  $(`#foul-blue-minor .fouls-global`).text(blueFouls.filter(foul => !foul.IsMajor).length);
  $(`#foul-blue-major .fouls-global`).text(blueFouls.filter(foul => foul.IsMajor).length);
  $(`#foul-red-minor .fouls-global`).text(redFouls.filter(foul => !foul.IsMajor).length);
  $(`#foul-red-major .fouls-global`).text(redFouls.filter(foul => foul.IsMajor).length);
};

const resetFoulCounts = function () {
  localFoulCounts["red-minor"] = 0;
  localFoulCounts["blue-minor"] = 0;
  localFoulCounts["red-major"] = 0;
  localFoulCounts["blue-major"] = 0;
  renderLocalFoulCounts();
};

const addFoul = function (alliance, isMajor) {
  const foulType = `${alliance}-${isMajor ? "major" : "minor"}`;
  localFoulCounts[foulType] += 1;
  renderLocalFoulCounts();
  websocket.send("addFoul", {Alliance: alliance, IsMajor: isMajor});
};

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

const resetLocalState = function () {
  committed = false;
  updateUIMode();
};

const updateUIMode = function () {
  $(".scoring-button").prop('disabled', !scoringAvailable);
  $(".scoring-tower-button").prop('disabled', !scoringAvailable);
  $("#commit").prop('disabled', !commitAvailable);
};

const endgameStatusNames = [
  "None",
  "Level 1",
  "Level 2",
  "Level 3",
];

const handleRealtimeScore = function (data) {
  let realtimeScore = alliance === "red" ? data.Red : data.Blue;
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

const handleRealtimeScoreCustom = function (data) {
  let realtimeScore = alliance === "red" ? data.Red : data.Blue;
  const score = realtimeScore.Score || {};
  const counts = score.counts || {};
  const boolStatusesMap = score.bool_statuses || {};
  const enumStatusesMap = score.enum_statuses || {};

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

const handleAutoTowerClick = function (teamPosition, autoTowerStatus) {
  websocket.send("autoTower", {TeamPosition: teamPosition, AutoTowerStatus: autoTowerStatus});
};
const handleEndgameClick = function (teamPosition, endgameTowerStatus) {
  websocket.send("endgame", {TeamPosition: teamPosition, EndgameTowerStatus: endgameTowerStatus});
};

const commitMatchScore = function () {
  websocket.send("commitMatch");
  committed = true;
  scoringAvailable = false;
  commitAvailable = false;
  updateUIMode();
};

const buildCustomUI = function (config) {
  const container = $("#tower-controls");
  if (!container.length) return;
  container.empty();

  const phases = [
    { key: "auto", title: "Autonomous" },
    { key: "teleop", title: "Teleoperated" },
    { key: "endgame", title: "Endgame" }
  ];

  phases.forEach(phase => {
    const phaseCounts = (config.scoring_counts || []).filter(sc =>
      (sc.phases || []).some(pp => pp.phase === phase.key)
    );
    const phaseStatuses = (config.statuses || []).filter(st =>
      (st.phases || []).some(pp => pp.phase === phase.key)
    );

    if (phaseCounts.length === 0 && phaseStatuses.length === 0) return;

    let html = `<section class="tower-section" id="${phase.key}-section" style="flex: none; height: auto;">`;
    html += `<h1>${phase.title}</h1>`;
    html += `<div style="display: flex; flex-direction: column; gap: 15px; width: 100%;">`;

    phaseCounts.forEach(sc => {
      html += `<div class="count-control" id="${sc.id}-${phase.key}" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">`;
      html += `<span class="count-label" style="font-size: 1.2rem; font-weight: bold;">${sc.display_name}</span>`;
      html += `<div style="display: flex; align-items: center; gap: 15px;">`;
      html += `<button class="scoring-button" onclick="adjustCount('${sc.id}', '${phase.key}', -1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">-</button>`;
      html += `<span class="count-value" id="${sc.id}-${phase.key}-count" style="font-size: 1.5rem; min-width: 30px; text-align: center;">0</span>`;
      html += `<button class="scoring-button" onclick="adjustCount('${sc.id}', '${phase.key}', 1);" ontouchstart disabled style="font-size: 1.5rem; width: 50px; height: 50px;">+</button>`;
      html += `</div></div>`;
    });

    phaseStatuses.forEach(st => {
      html += `<div class="status-control" id="${st.id}-status" style="display: flex; align-items: center; justify-content: space-between; padding: 10px; background: rgba(255,255,255,0.05); border-radius: 4px;">`;
      html += `<span class="status-label" style="font-size: 1.2rem; font-weight: bold;">${st.display_name}</span>`;
      html += `<div style="display: flex; gap: 8px;">`;

      for (let i = 0; i < 3; i++) {
        const teamClass = `team-${i + 1}`;
        if (st.values && st.values.length > 0) {
          const defaultLabel = st.values[0] ? st.values[0].display_name : "None";
          html += `<div class="${teamClass}" style="display: flex; flex-direction: column; align-items: center; gap: 4px;">`;
          html += `<span class="team-num" style="font-size: 0.85rem; font-weight: bold;"></span>`;
          html += `<button class="scoring-button status-toggle" id="${st.id}-${i}" onclick="cycleEnumStatus('${st.id}', ${i});" ontouchstart disabled style="width: 80px; height: 45px;">${defaultLabel}</button>`;
          html += `</div>`;
        } else {
          html += `<div class="${teamClass}" style="display: flex; flex-direction: column; align-items: center; gap: 4px;">`;
          html += `<span class="team-num" style="font-size: 0.85rem; font-weight: bold;"></span>`;
          html += `<button class="scoring-button status-toggle" id="${st.id}-${i}" onclick="toggleBoolStatus('${st.id}', ${i});" ontouchstart disabled style="width: 60px; height: 45px;"></button>`;
          html += `</div>`;
        }
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
  alliance = position;
  $(".container").attr("data-alliance", alliance);
  resetLocalState();

  $.getJSON("/api/game_config")
    .done(function (config) {
      if (config && config.game && config.game.name) {
        IS_CUSTOM_GAME_MODE = true;
        window.gameConfig = config;
        buildCustomUI(config);
      }
    })
    .always(function () {
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
