package main

type GameYAML struct {
	Game               GameInfo       `yaml:"game"`
	Fouls              FoulConfig     `yaml:"fouls"`
	GamePieces         []GamePiece    `yaml:"game_pieces"`
	DisplayGroups      []DisplayGroup `yaml:"display_groups"`
	ScoringCounts      []ScoringCount `yaml:"scoring_counts"`
	Statuses           []Status       `yaml:"statuses"`
	RPs                []RankingPoint `yaml:"ranking_points"`
	RankingTiebreakers []Tiebreaker   `yaml:"ranking_tiebreakers"`
	PlayoffTiebreakers []Tiebreaker   `yaml:"playoff_tiebreakers"`
}

type GameInfo struct {
	Name string `yaml:"name"`
}

type FoulConfig struct {
	MinorFoulPoints int `yaml:"minor_foul_points"`
	MajorFoulPoints int `yaml:"major_foul_points"`
}

type GamePiece struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
}

// DisplayGroup is a named bucket used only to roll up Elements for the audience display
// (live counters and final breakdown). Independent of GamePiece, which tracks real piece
// identity and is never used for display rollups directly.
type DisplayGroup struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
}

type ScoringCount struct {
	ID           string         `yaml:"id"`
	DisplayName  string         `yaml:"display_name"`
	GamePiece    string         `yaml:"game_piece"`    // optional, FK into game_pieces
	DisplayGroup string         `yaml:"display_group"` // optional, FK into display_groups
	Phases       []ElementPhase `yaml:"phases"`
}

// ElementPhase declares that an Element is scored during the given phase, worth Points
// each time. An Element with multiple ElementPhase entries generates one Count field per
// phase (e.g. AutoFooCount and TeleopFooCount), each accumulating independently.
type ElementPhase struct {
	Phase  string `yaml:"phase"` // "auto" | "teleop" | "endgame"
	Points int    `yaml:"points"`
}

// HasPhase reports whether the ScoringCount declares the given phase.
func (e *ScoringCount) HasPhase(phase string) bool {
	for _, ep := range e.Phases {
		if ep.Phase == phase {
			return true
		}
	}
	return false
}

// Status declares a per-robot status flag. Phases must have exactly one entry, phase "auto" or
// "endgame" (teleop not supported — see CUSTOM_GAMES.md for why). Phases[0].Points is the bool-
// status point value (sugar for an implicit {false: 0, true: Points} values list); it's unused
// when Values is set, since each StatusValue then carries its own per-state points instead.
type Status struct {
	ID          string         `yaml:"id"`
	DisplayName string         `yaml:"display_name"`
	Phases      []ElementPhase `yaml:"phases"`
	Values      []StatusValue  `yaml:"values"`
}

type StatusValue struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
	Points      int    `yaml:"points"`
}

type RankingPoint struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
	LogicFunc   string `yaml:"logic_func"`
}

type Tiebreaker struct {
	Metric string `yaml:"metric"`
}
