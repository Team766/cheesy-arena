// Copyright 2024 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
//go:build !custom

package field

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"math"
	"time"
)

// The game-specific parts of the team sign rear text: which live totals the 2026 game shows, and
// the Hub shift-aware period indicator. The custom-game build supplies generic versions of these
// three functions in team_sign_custom.go; everything else in team_sign.go is shared.

// Returns the in-match rear text for the team number display that is common to the whole given alliance.
func generateInMatchTeamRearText(arena *Arena, isRed bool, countdown string, currentTime time.Time) string {
	allianceScores := generateTeamSignAllianceScores(arena, isRed)
	periodText := generateTeamSignPeriodText(arena, currentTime)
	if arena.CurrentMatch.Type == model.Playoff {
		return formatTeamSignRearText(fmt.Sprintf("%s %s %s", periodText, allianceScores, countdown))
	}

	var realtimeScore, opponentRealtimeScore *RealtimeScore
	if isRed {
		realtimeScore = arena.RedRealtimeScore
		opponentRealtimeScore = arena.BlueRealtimeScore
	} else {
		realtimeScore = arena.BlueRealtimeScore
		opponentRealtimeScore = arena.RedRealtimeScore
	}
	scoreSummary := realtimeScore.CurrentScore.Summarize(&opponentRealtimeScore.CurrentScore)
	numFuel := scoreSummary.NumFuel - scoreSummary.NumFuelPostMatch
	numFuelGoal := game.EnergizedBonusThreshold
	if numFuel >= game.EnergizedBonusThreshold {
		numFuelGoal = game.SuperchargedBonusThreshold
	}
	return formatTeamSignRearText(
		fmt.Sprintf("%s %3d/%d %2d %s", periodText, numFuel, numFuelGoal, scoreSummary.AutoTowerPoints, countdown),
	)
}

// Returns the live score string for the given alliance, excluding post-match points.
func generateTeamSignAllianceScores(arena *Arena, isRed bool) string {
	var realtimeScore, opponentRealtimeScore *RealtimeScore
	var formatString string
	if isRed {
		realtimeScore = arena.RedRealtimeScore
		opponentRealtimeScore = arena.BlueRealtimeScore
		formatString = "R%03d-B%03d"
	} else {
		realtimeScore = arena.BlueRealtimeScore
		opponentRealtimeScore = arena.RedRealtimeScore
		formatString = "B%03d-R%03d"
	}
	scoreSummary := realtimeScore.CurrentScore.Summarize(&opponentRealtimeScore.CurrentScore)
	scoreTotal := scoreSummary.Score - scoreSummary.PostMatchPoints
	opponentScoreSummary := opponentRealtimeScore.CurrentScore.Summarize(&realtimeScore.CurrentScore)
	opponentScoreTotal := opponentScoreSummary.Score - opponentScoreSummary.PostMatchPoints
	return fmt.Sprintf(formatString, scoreTotal, opponentScoreTotal)
}

// Returns the match period indicator and remaining time shown at the start of the team sign rear text.
func generateTeamSignPeriodText(arena *Arena, currentTime time.Time) string {
	shift, remaining, _, ok := arena.RedRealtimeScore.CurrentScore.Hub.GetCurrentShiftTiming(
		arena.MatchStartTime, currentTime,
	)
	periodPrefix := generateTeamSignPeriodPrefix(arena, shift)
	if !ok {
		return periodPrefix + "00"
	}
	return fmt.Sprintf("%s%02d", periodPrefix, int(math.Ceil(remaining.Seconds())))
}

// Returns the sign character corresponding to the given Hub shift.
func generateTeamSignPeriodPrefix(arena *Arena, shift game.Shift) string {
	switch shift {
	case game.ShiftAuto:
		return "A"
	case game.ShiftTransition:
		return "T"
	case game.Shift1, game.Shift3:
		if arena.RedRealtimeScore.CurrentScore.Hub.WonAuto {
			return "B"
		}
		return "R"
	case game.Shift2, game.Shift4:
		if arena.RedRealtimeScore.CurrentScore.Hub.WonAuto {
			return "R"
		}
		return "B"
	case game.ShiftEndgame:
		return "E"
	default:
		return "E"
	}
}
