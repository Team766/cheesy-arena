// Copyright 2014 Team 254. All Rights Reserved.
// Author: nick@team254.com (Nick Eyre)
// Author: pat@patfairbank.com (Patrick Fairbank)
//
// Client-side methods for the rankings display.

var websocket;
var initialDwellMs = 3000;  // How long the display waits upon initial load before scrolling.
var scrollMsPerRow;  // How long in milliseconds it takes to scroll a height of one row.
var staticUpdateIntervalMs = 10000;  // How long between updates if not scrolling.
var standingsTemplate = Handlebars.compile($("#standingsTemplate").html());
var rankingsData;
var prevHighestPlayedMatch;

// Labels for the ranking tiebreaker metrics that aren't a scoring group, count or status id.
const builtInTiebreakerLabels = {
  "auto_points": "Auto",
  "teleop_points": "Teleop",
  "endgame_points": "Endgame",
  "total_points": "Total",
};

// Returns the value of one tiebreaker metric, defaulting to zero for a team that hasn't played yet
// and therefore has no tiebreaker map at all.
Handlebars.registerHelper("tiebreaker", function (tiebreakers, metric) {
  return (tiebreakers && tiebreakers[metric]) || 0;
});

// Custom game mode: the columns shown between RP and W-L-T, one per configured ranking tiebreaker.
var customRankingColumns = function (config) {
  return (config.ranking_tiebreakers || []).map(function (tiebreaker) {
    const metric = tiebreaker.metric;
    let label = builtInTiebreakerLabels[metric];
    if (label === undefined) {
      const element = (config.scoring_groups || []).find(sg => sg.id === metric) ||
        (config.scoring_counts || []).find(sc => sc.id === metric) ||
        (config.statuses || []).find(st => st.id === metric);
      label = element ? element.display_name : metric;
    }
    return {metric: metric, label: label};
  });
};

// Custom game mode: replaces the game-specific header cells and recompiles the row template, since
// which point columns exist is only known once the game configuration has been fetched.
var buildCustomStandings = function (config) {
  let headerHtml =
    '<td class="team-field">Rank</td>' +
    '<td class="team-field">Team</td>' +
    '<td class="team-nickname">Name</td>' +
    '<td class="team-field">RP</td>';
  let rowHtml =
    '<td class="team-field">{{../Iteration}} {{this.Rank}}</td>' +
    '<td class="team-field">{{this.TeamId}}</td>' +
    '<td class="team-nickname">{{this.Nickname}}</td>' +
    '<td class="team-field">{{this.RankingPoints}}</td>';

  customRankingColumns(config).forEach(function (column) {
    headerHtml += `<td class="team-field">${column.label}</td>`;
    rowHtml += `<td class="team-field">{{tiebreaker this.Tiebreakers "${column.metric}"}}</td>`;
  });

  headerHtml +=
    '<td class="team-field">W-L-T</td>' +
    '<td class="team-field">DQ</td>' +
    '<td class="team-field">Played</td>';
  rowHtml +=
    '<td class="team-field">{{this.Wins}}-{{this.Losses}}-{{this.Ties}}</td>' +
    '<td class="team-field">{{this.Disqualifications}}</td>' +
    '<td class="team-field">{{this.Played}}</td>';

  $("#standingsHeaderRow").html(headerHtml);
  standingsTemplate = Handlebars.compile(`<tbody>{{#each Rankings}}<tr>${rowHtml}</tr>{{/each}}</tbody>`);
};

// Loads the JSON rankings data from the event server.
var getRankingsData = function (callback) {
  $.getJSON("/api/rankings", function (data) {
    rankingsData = data;
    if (callback) {
      callback(data);
    }
  });
};

// Updates the rankings in place and initiates scrolling if they are long enough to require it.
var updateStaticRankings = function () {
  getRankingsData(function () {
    var rankingsHtml = standingsTemplate(rankingsData);
    $("#rankings2").html(rankingsHtml);
    $("#scroller").css("transform", "translate(0px, -2px);");
    prevHighestPlayedMatch = rankingsData.HighestPlayedMatch;
    setHighestPlayedMatch(rankingsData.HighestPlayedMatch);
    if ($("#rankings2").height() > $("#container").height()) {
      // Initiate scrolling.
      setTimeout(cycleRankings, initialDwellMs);
    } else {
      // Rankings are too short; just update in place.
      setTimeout(updateStaticRankings, staticUpdateIntervalMs);
    }
  });
};

// Seamlessly copies the newer table contents to the older one, resets the scrolling, and loads new data.
var cycleRankings = function () {
  // Overwrite the top data with the bottom data and reset the scrolling back up to the top of the top table.
  $("#rankings1").html($("#rankings2").html());
  $("#scroller").css({transform: "translate(0px, -1px);"});

  // Load new data into the now out-of-sight bottom table.
  var rankingsHtml = standingsTemplate(rankingsData);
  $("#rankings2").html(rankingsHtml);

  // Delay updating the "Standings as of" message by one cycle because the tables are always one cycle behind
  // the data loading.
  setHighestPlayedMatch(prevHighestPlayedMatch);
  prevHighestPlayedMatch = rankingsData.HighestPlayedMatch;

  if ($("#rankings1").height() > $("#container").height()) {
    // Kick off another scrolling animation.
    var scrollDistance = $("#rankings1").height() + parseInt($("#rankings1").css("border-bottom-width"));
    var scrollTime = scrollMsPerRow * $("#rankings1 tr").length;
    $("#scroller").transition({y: -scrollDistance}, scrollTime, "linear", cycleRankings);

    // Set the data to be reloaded two seconds before the scrolling terminates.
    var reloadDataTime = Math.max(0, scrollTime - 2000);
    setTimeout(getRankingsData, reloadDataTime);
  } else {
    // The rankings got shorter for whatever reason, so revert to static updating.
    setTimeout(updateStaticRankings, staticUpdateIntervalMs);
  }
};

// Updates the "Standings as of" message with the given value, or blanks it out if there is no data yet.
var setHighestPlayedMatch = function (highestPlayedMatch) {
  if (highestPlayedMatch === "") {
    $("#highestPlayedMatch").text("");
  } else {
    $("#highestPlayedMatch").text("Standings as of " + highestPlayedMatch);
  }
};

// Handles a websocket message to update the event status message.
var handleEventStatus = function (data) {
  $("#earlyLateMessage").text(data.EarlyLateMessage);
};

$(function () {
  // Read the configuration for this display from the URL query string.
  var urlParams = new URLSearchParams(window.location.search);
  scrollMsPerRow = urlParams.get("scrollMsPerRow");

  // Set up the websocket back to the server. Used only for remote forcing of reloads.
  websocket = new CheesyWebsocket("/displays/rankings/websocket", {
    eventStatus: function (event) {
      handleEventStatus(event.data);
    },
  });

  // Fetch the game configuration before the first render so that the columns match the game being
  // played; the endpoint 404s in the stock build, in which case .done() is skipped and the
  // server-rendered FRC columns are kept.
  $.getJSON("/api/game_config")
    .done(function (config) {
      if (config && config.game && config.game.name) {
        buildCustomStandings(config);
      }
    })
    .always(function () {
      updateStaticRankings();
    });
});
