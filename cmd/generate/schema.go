package main

type GameYAML struct {
	Game               GameInfo       `yaml:"game"`
	Fouls              FoulConfig     `yaml:"fouls"`
	GamePieces         []GamePiece    `yaml:"game_pieces"`
	ScoringCounts      []Element      `yaml:"scoring_counts"`
	EndgameCounts      []Element      `yaml:"endgame_counts"`
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

type Element struct {
	ID           string `yaml:"id"`
	DisplayName  string `yaml:"display_name"`
	GamePiece    string `yaml:"game_piece"` // optional
	Phase        string `yaml:"phase"`      // "auto" | "teleop" | "both"
	Points       int    `yaml:"points"`     // shorthand for single-phase
	PointsAuto   int    `yaml:"points_auto"`
	PointsTeleop int    `yaml:"points_teleop"`
	Group        string `yaml:"group"` // optional UI grouping hint
}

type Status struct {
	ID          string        `yaml:"id"`
	DisplayName string        `yaml:"display_name"`
	Phase       string        `yaml:"phase"` // "auto" | "endgame"
	Points      int           `yaml:"points"`
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
