// Copyright 2023 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side logic for the referee interface.

var websocket;
let redFoulsHashCode = 0;
let blueFoulsHashCode = 0;
let scoreIsReady = false;
let isPostMatch = false;

let IS_CUSTOM_GAME_MODE = false;
window.gameConfig = null;

const addFoul = function (alliance, isMajor) {
  websocket.send("addFoul", {Alliance: alliance, IsMajor: isMajor});
};

const toggleFoulType = function (alliance, index) {
  websocket.send("toggleFoulType", {Alliance: alliance, Index: index});
};

const updateFoulTeam = function (alliance, index, teamId) {
  websocket.send("updateFoulTeam", {Alliance: alliance, Index: index, TeamId: teamId});
};

const updateFoulRule = function (alliance, index, ruleId) {
  websocket.send("updateFoulRule", {Alliance: alliance, Index: index, RuleId: ruleId});
};

var deleteFoul = function (alliance, index) {
  websocket.send("deleteFoul", {Alliance: alliance, Index: index});
};

var cycleCard = function (cardButton) {
  if (isPostMatch) {
    const currentCard = $(cardButton).attr("data-card");
    const hasOldYellowCard = $(cardButton).attr("data-old-yellow-card") === "true";
    let newCard = "";
    if (currentCard === "" && hasOldYellowCard) {
      newCard = "red";
    } else if (currentCard === "") {
      newCard = "yellow";
    } else if (currentCard === "yellow") {
      newCard = "red";
    }
    websocket.send(
      "card",
      {Alliance: $(cardButton).attr("data-alliance"), TeamId: parseInt($(cardButton).attr("data-team")), Card: newCard}
    );
    $(cardButton).attr("data-card", newCard);
    return;
  }

  const isDisabled = $(cardButton).hasClass("bypassed-status");
  const team = $(cardButton).attr("data-team");
  $("#confirmBypassTitle").text(`${isDisabled ? "Enable" : "Disable"} ${team}?`);
  $("#confirmBypassAction").text(isDisabled ? "Enable" : "Disable");
  $("#confirmBypass").attr("data-station", $(cardButton).attr("data-station")?.toUpperCase());

  if (team === "0") {
    toggleBypass();
  } else {
    $("#confirmBypass").modal("show");
  }
};

const toggleBypass = function () {
  const station = $("#confirmBypass").attr("data-station");
  websocket.send("toggleBypass", station);
};

var signalVolunteers = function () {
  websocket.send("signalVolunteers");
};

var signalReset = function () {
  websocket.send("signalReset");
};

var confirmCommit = function () {
  if (scoreIsReady) {
    commitAndPost();
    return;
  }
  $("#confirmCommit").modal("show");
};

var commitAndPost = function () {
  websocket.send("commitAndPost");
};

var handleMatchLoad = function (data) {
  $("#matchName").text(data.Match.LongName);

  setTeamCard("red", 1, data.Teams["R1"]);
  setTeamCard("red", 2, data.Teams["R2"]);
  setTeamCard("red", 3, data.Teams["R3"]);
  setTeamCard("blue", 1, data.Teams["B1"]);
  setTeamCard("blue", 2, data.Teams["B2"]);
  setTeamCard("blue", 3, data.Teams["B3"]);

  $("#redScoreSummary .team-1").text(data.Teams["R1"]?.Id || "");
  $("#redScoreSummary .team-2").text(data.Teams["R2"]?.Id || "");
  $("#redScoreSummary .team-3").text(data.Teams["R3"]?.Id || "");
  $("#blueScoreSummary .team-1").text(data.Teams["B1"]?.Id || "");
  $("#blueScoreSummary .team-2").text(data.Teams["B2"]?.Id || "");
  $("#blueScoreSummary .team-3").text(data.Teams["B3"]?.Id || "");
};

const handleMatchTime = function (data) {
  isPostMatch = matchStates[data.MatchState] === "POST_MATCH";
  $(".control-button").attr("data-enabled", isPostMatch);

  let title = "Red/Yellow Cards";
  if (!isPostMatch) {
    title = matchStates[data.MatchState] === "PRE_MATCH" ? "Bypass" : "Disable";
  }
  $("#teamTitle").text(title);
};

const towerStatusNames = [
  "None",
  "Level 1",
  "Level 2",
  "Level 3",
];

const setTowerStatus = function (selector, status) {
  $(selector).text(towerStatusNames[status]);
  $(selector).attr("data-status", status);
};

const updateScoreSummaryCustom = function (scoreRoot, score) {
  if (!window.gameConfig) return;
  const counts = score.counts || {};
  const boolStatusesMap = score.bool_statuses || {};
  const enumStatusesMap = score.enum_statuses || {};

  (window.gameConfig.statuses || []).forEach(st => {
    if (st.values && st.values.length > 0) {
      const arr = enumStatusesMap[st.id] || [0, 0, 0];
      for (let i = 0; i < 3; i++) {
        const valIdx = arr[i] || 0;
        const valName = st.values[valIdx] ? st.values[valIdx].display_name : "-";
        $(`#${scoreRoot} .team-${i + 1}-${st.id}`)
          .text(valName)
          .attr("data-active", valIdx !== 0);
      }
    } else {
      const arr = boolStatusesMap[st.id] || [false, false, false];
      for (let i = 0; i < 3; i++) {
        const val = !!arr[i];
        $(`#${scoreRoot} .team-${i + 1}-${st.id}`)
          .text(val ? "Yes" : "No")
          .attr("data-active", val);
      }
    }
  });

  const phases = ["auto", "teleop", "endgame"];
  phases.forEach(phase => {
    const phaseCounts = (window.gameConfig.scoring_counts || []).filter(sc =>
      (sc.phases || []).some(pp => pp.phase === phase)
    );
    if (phaseCounts.length > 0) {
      const totals = phaseCounts.map(sc => {
        const key = `${sc.id}_${phase}`;
        return counts[key] !== undefined ? counts[key] : 0;
      });
      $(`#${scoreRoot} .phase-${phase}`).text(totals.join(" / "));
    }
  });
};

const handleRealtimeScore = function (data) {
  for (const [teamId, card] of Object.entries(Object.assign(data.RedCards, data.BlueCards))) {
    $(`[data-team="${teamId}"]`).attr("data-card", card);
  }

  const newRedFoulsHashCode = hashObject(data.Red.Score.Fouls);
  const newBlueFoulsHashCode = hashObject(data.Blue.Score.Fouls);
  if (newRedFoulsHashCode !== redFoulsHashCode || newBlueFoulsHashCode !== blueFoulsHashCode) {
    redFoulsHashCode = newRedFoulsHashCode;
    blueFoulsHashCode = newBlueFoulsHashCode;
    fetch("/panels/referee/foul_list")
      .then(response => response.text())
      .then(svg => $("#foulList").html(svg));
  }

  for (alliance of ["red", "blue"]) {
    let score = alliance === "red" ? data.Red.Score : data.Blue.Score;
    let scoreRoot = `${alliance}ScoreSummary`;
    if (IS_CUSTOM_GAME_MODE) {
      updateScoreSummaryCustom(scoreRoot, score);
    } else {
      setTowerStatus(`#${scoreRoot} .team-1-auto-tower`, score.AutoTowerStatuses[0]);
      setTowerStatus(`#${scoreRoot} .team-2-auto-tower`, score.AutoTowerStatuses[1]);
      setTowerStatus(`#${scoreRoot} .team-3-auto-tower`, score.AutoTowerStatuses[2]);
      setTowerStatus(`#${scoreRoot} .team-1-endgame-tower`, score.EndgameTowerStatuses[0]);
      setTowerStatus(`#${scoreRoot} .team-2-endgame-tower`, score.EndgameTowerStatuses[1]);
      setTowerStatus(`#${scoreRoot} .team-3-endgame-tower`, score.EndgameTowerStatuses[2]);
    }
  }
};

const handleScoringStatus = function (data) {
  if (data.RefereeScoreReady) {
    $("#commitButton").attr("data-enabled", false);
  }
  updateScoreStatus(data, "red", "#redScoreStatus", "Red");
  updateScoreStatus(data, "blue", "#blueScoreStatus", "Blue");

  scoreIsReady = Object.values(data.PositionStatuses).every(status => status.Ready);
  if (scoreIsReady) {
    $("#commitButton").removeClass("disabled");
  } else {
    $("#commitButton").addClass("disabled");
  }
};

const handleArenaStatus = function (data) {
  setTeamBypassedStatus("red1", data.AllianceStations["R1"]?.Bypass);
  setTeamBypassedStatus("red2", data.AllianceStations["R2"]?.Bypass);
  setTeamBypassedStatus("red3", data.AllianceStations["R3"]?.Bypass);
  setTeamBypassedStatus("blue1", data.AllianceStations["B1"]?.Bypass);
  setTeamBypassedStatus("blue2", data.AllianceStations["B2"]?.Bypass);
  setTeamBypassedStatus("blue3", data.AllianceStations["B3"]?.Bypass);
};

const setTeamBypassedStatus = function (station, bypassed) {
  const cardButton = $(`#${station}Card`);
  cardButton.toggleClass("bypassed-status", bypassed && !isPostMatch);
};

const updateScoreStatus = function (data, position, element, displayName) {
  const status = data.PositionStatuses[position];
  $(element).text(`${displayName} ${status.NumPanelsReady}/${status.NumPanels}`);
  $(element).attr("data-present", status.NumPanels > 0);
  $(element).attr("data-ready", status.Ready);
};

const setTeamCard = function (alliance, position, team) {
  const cardButton = $(`#${alliance}${position}Card`);
  if (team === null) {
    cardButton.text("-");
    cardButton.attr("data-team", 0);
    cardButton.attr("data-old-yellow-card", "");
  } else {
    cardButton.text(team.Id);
    cardButton.attr("data-team", team.Id);
    cardButton.attr("data-old-yellow-card", team.YellowCard);
  }
  cardButton.attr("data-card", "");
};

const hashObject = function (object) {
  const s = JSON.stringify(object);
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = Math.imul(31, h) + s.charCodeAt(i) | 0;
  }
  return h;
};

const buildRefScoreSummaryUI = function (config) {
  ["redScoreSummary", "blueScoreSummary"].forEach(id => {
    const container = $(`#${id}-dynamic-breakdown`);
    if (!container.length) return;
    container.empty();

    const phases = [
      { key: "auto", title: "Auto" },
      { key: "teleop", title: "Teleop" },
      { key: "endgame", title: "Endgame" }
    ];

    let html = "";
    phases.forEach(phase => {
      const phaseCounts = (config.scoring_counts || []).filter(sc =>
        (sc.phases || []).some(pp => pp.phase === phase.key)
      );
      if (phaseCounts.length > 0) {
        const zeros = phaseCounts.map(() => "0").join(" / ");
        html += `<div class="label">${phase.title}</div>`;
        html += `<div class="wide-row count-total phase-${phase.key}">${zeros}</div>`;
      }
    });

    (config.statuses || []).forEach(st => {
      html += `<div class="label">${st.display_name}</div>`;
      for (let i = 1; i <= 3; i++) {
        html += `<div class="status-badge team-${i}-${st.id}" data-active="false">-</div>`;
      }
    });

    container.append(html);
  });
};

$(function () {
  var urlParams = new URLSearchParams(window.location.search);
  $(".headRef-dependent").attr("data-hr", urlParams.get("hr"));

  $.getJSON("/api/game_config")
    .done(function (config) {
      if (config && config.game && config.game.name) {
        IS_CUSTOM_GAME_MODE = true;
        window.gameConfig = config;
        buildRefScoreSummaryUI(config);
      }
    })
    .always(function () {
      websocket = new CheesyWebsocket("/panels/referee/websocket", {
        matchLoad: function (event) {
          handleMatchLoad(event.data);
        },
        matchTime: function (event) {
          handleMatchTime(event.data);
        },
        realtimeScore: function (event) {
          handleRealtimeScore(event.data);
        },
        scoringStatus: function (event) {
          handleScoringStatus(event.data);
        },
        arenaStatus: function (event) {
          handleArenaStatus(event.data);
        },
      });
    });
});
