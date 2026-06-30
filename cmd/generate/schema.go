package main

type GameYAML struct {
	Game               GameInfo       `yaml:"game"`
	Fouls              FoulConfig     `yaml:"fouls"`
	GamePieces         []GamePiece    `yaml:"game_pieces"`
	ScoringGroups      []ScoringGroup `yaml:"scoring_groups"`
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

// ScoringGroup is a named rollup of scoring counts. Its member counts' points are summed into one
// ScoreSummary field (summary.<Group>Points) and shown together on the audience display, and the
// group is the unit tiebreakers reference. Separate from GamePiece, which tracks real piece
// identity, not how scoring rolls up.
type ScoringGroup struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
}

type ScoringCount struct {
	ID           string        `yaml:"id"`
	DisplayName  string        `yaml:"display_name"`
	GamePiece    string        `yaml:"game_piece"`    // required; names a game_pieces id (piece identity, not a rollup)
	ScoringGroup string        `yaml:"scoring_group"` // optional; names a scoring_groups id (the rollup bucket)
	Phases       []PhasePoints `yaml:"phases"`
}

// PhasePoints declares that a scoring count (or status) is scored during the given phase, worth
// Points each time. A scoring count with multiple PhasePoints entries generates one Count field per
// phase (e.g. AutoFooCount and TeleopFooCount), each accumulating independently.
type PhasePoints struct {
	Phase  string `yaml:"phase"` // "auto" | "teleop" | "endgame"
	Points int    `yaml:"points"`
}

// Status declares a per-robot status flag. Phases must have exactly one entry, phase "auto" or
// "endgame" (teleop not supported — see CUSTOM_GAMES.md for why). Phases[0].Points is the bool-
// status point value (sugar for an implicit {false: 0, true: Points} values list); it's unused
// when Values is set, since each StatusValue then carries its own per-state points instead.
type Status struct {
	ID          string        `yaml:"id"`
	DisplayName string        `yaml:"display_name"`
	Phases      []PhasePoints `yaml:"phases"`
	Values      []StatusValue `yaml:"values"`
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
