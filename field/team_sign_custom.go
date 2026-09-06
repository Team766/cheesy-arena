// Copyright 2024 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)
//
//go:build custom

package field

import (
	"fmt"
	"time"
)

// The game-specific parts of the team sign rear text, in their game-agnostic form. A custom game
// has no equivalent of the 2026 game's fuel counters or Hub shifts, so the rear text shows the
// alliance scores in every match type and the period indicator follows the arena's match state.
// Everything else about the signs — packet format, colors, blinking, station messages — is shared
// with the standard build in team_sign.go.

// Returns the in-match rear text for the team number display that is common to the whole given alliance.
func generateInMatchTeamRearText(arena *Arena, isRed bool, countdown string, currentTime time.Time) string {
	allianceScores := generateTeamSignAllianceScores(arena, isRed)
	periodText := generateTeamSignPeriodText(arena, currentTime)
	return formatTeamSignRearText(fmt.Sprintf("%s %s %s", periodText, allianceScores, countdown))
}

// Returns the live score string for the given alliance.
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
	opponentScoreSummary := opponentRealtimeScore.CurrentScore.Summarize(&realtimeScore.CurrentScore)
	return fmt.Sprintf(formatString, scoreSummary.Score, opponentScoreSummary.Score)
}

// Returns the match period indicator shown at the start of the team sign rear text.
func generateTeamSignPeriodText(arena *Arena, currentTime time.Time) string {
	switch arena.MatchState {
	case AutoPeriod:
		return "A"
	case TeleopPeriod:
		return "T"
	}
	return "E"
}
