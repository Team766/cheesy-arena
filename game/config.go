package game

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
	"strings"
	"sync"
)

type GameYAML struct {
	Game               GameInfo       `yaml:"game" json:"game"`
	Fouls              FoulConfig     `yaml:"fouls" json:"fouls"`
	GamePieces         []GamePiece    `yaml:"game_pieces" json:"game_pieces"`
	ScoringGroups      []ScoringGroup `yaml:"scoring_groups" json:"scoring_groups"`
	ScoringCounts      []ScoringCount `yaml:"scoring_counts" json:"scoring_counts"`
	Statuses           []Status       `yaml:"statuses" json:"statuses"`
	RPs                []RankingPoint `yaml:"ranking_points" json:"ranking_points"`
	RankingTiebreakers []Tiebreaker   `yaml:"ranking_tiebreakers" json:"ranking_tiebreakers"`
	PlayoffTiebreakers []Tiebreaker   `yaml:"playoff_tiebreakers" json:"playoff_tiebreakers"`

	// Lookup indexes, built by buildIndexes when the config is activated. Unexported so that they
	// are neither serialized nor settable from YAML.
	countByID  map[string]*ScoringCount
	statusByID map[string]*Status
}

type GameInfo struct {
	Name string `yaml:"name" json:"name"`
}

type FoulConfig struct {
	MinorFoulPoints int `yaml:"minor_foul_points" json:"minor_foul_points"`
	MajorFoulPoints int `yaml:"major_foul_points" json:"major_foul_points"`
}

type GamePiece struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"display_name" json:"display_name"`
}

type ScoringGroup struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"display_name" json:"display_name"`
}

type ScoringCount struct {
	ID           string        `yaml:"id" json:"id"`
	DisplayName  string        `yaml:"display_name" json:"display_name"`
	GamePiece    string        `yaml:"game_piece" json:"game_piece"`
	ScoringGroup string        `yaml:"scoring_group,omitempty" json:"scoring_group,omitempty"`
	Scorer       string        `yaml:"scorer,omitempty" json:"scorer,omitempty"`
	Phases       []PhasePoints `yaml:"phases" json:"phases"`
}

type PhasePoints struct {
	Phase  string `yaml:"phase" json:"phase"`
	Points int    `yaml:"points" json:"points"`
}

type Status struct {
	ID          string        `yaml:"id" json:"id"`
	DisplayName string        `yaml:"display_name" json:"display_name"`
	Scorer      string        `yaml:"scorer,omitempty" json:"scorer,omitempty"`
	Phases      []PhasePoints `yaml:"phases" json:"phases"`
	Values      []StatusValue `yaml:"values,omitempty" json:"values,omitempty"`
}

// Scorer hint values: which scoring panel (near or far side of the field) is responsible for an
// element. Empty means every panel shows it.
const (
	ScorerNear = "near"
	ScorerFar  = "far"
)

// ValidScorers lists the accepted values of the optional scorer field.
var ValidScorers = map[string]bool{"": true, ScorerNear: true, ScorerFar: true}

type StatusValue struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"display_name" json:"display_name"`
	Points      int    `yaml:"points" json:"points"`
}

type RankingPoint struct {
	ID          string `yaml:"id" json:"id"`
	DisplayName string `yaml:"display_name" json:"display_name"`
	LogicFunc   string `yaml:"logic_func" json:"logic_func"`
}

type Tiebreaker struct {
	Metric string `yaml:"metric" json:"metric"`
}

var (
	activeConfigMu sync.RWMutex
	activeConfig   *GameYAML
)

var goIdentRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var validElementPhases = map[string]bool{"auto": true, "teleop": true, "endgame": true}
var validStatusPhases = map[string]bool{"auto": true, "endgame": true}

// builtinMetricIDs are the summary metrics that exist for every config, in the order MetricIDs
// reports them.
var builtinMetricIDs = []string{"auto_points", "teleop_points", "endgame_points", "total_points"}

// builtinMetricLabels are the display names for the built-in metrics.
var builtinMetricLabels = map[string]string{
	"auto_points":    "Auto Points",
	"teleop_points":  "Teleop Points",
	"endgame_points": "Endgame Points",
	"total_points":   "Total Points",
}

// reservedIDs are identifiers that a config may not use for a scoring group, scoring count or
// status: either they name a built-in metric, or they name a ScoreSummary/RankingFields field that
// would be ambiguous with one.
var reservedIDs = map[string]bool{
	"auto_points":    true,
	"teleop_points":  true,
	"endgame_points": true,
	"total_points":   true,
	"score":          true,
	"match_points":   true,
	"foul_points":    true,
	"ranking_points": true,
}

func GetActiveConfig() *GameYAML {
	activeConfigMu.RLock()
	defer activeConfigMu.RUnlock()
	return activeConfig
}

func SetActiveConfig(cfg *GameYAML) {
	if cfg != nil {
		cfg.buildIndexes()
	}
	activeConfigMu.Lock()
	activeConfig = cfg
	activeConfigMu.Unlock()
	applyGameConfigConstants(cfg)
}

// buildIndexes populates the by-id lookup maps so that the hot-path score mutators don't have to
// linear-scan the config on every button press.
func (cfg *GameYAML) buildIndexes() {
	cfg.countByID = make(map[string]*ScoringCount, len(cfg.ScoringCounts))
	for i := range cfg.ScoringCounts {
		cfg.countByID[cfg.ScoringCounts[i].ID] = &cfg.ScoringCounts[i]
	}
	cfg.statusByID = make(map[string]*Status, len(cfg.Statuses))
	for i := range cfg.Statuses {
		cfg.statusByID[cfg.Statuses[i].ID] = &cfg.Statuses[i]
	}
}

// Count returns the scoring count with the given id, or nil.
func (cfg *GameYAML) Count(id string) *ScoringCount {
	if cfg == nil {
		return nil
	}
	if cfg.countByID == nil {
		cfg.buildIndexes()
	}
	return cfg.countByID[id]
}

// Status returns the status with the given id, or nil.
func (cfg *GameYAML) Status(id string) *Status {
	if cfg == nil {
		return nil
	}
	if cfg.statusByID == nil {
		cfg.buildIndexes()
	}
	return cfg.statusByID[id]
}

// Bucket returns the id of the summary bucket a scoring count contributes to: its scoring group if
// it has one, otherwise its own id.
func (sc *ScoringCount) Bucket() string {
	if sc.ScoringGroup != "" {
		return sc.ScoringGroup
	}
	return sc.ID
}

// MetricIDs returns every metric name that a tiebreaker may reference and that
// ScoreSummary.GetMetric understands: the built-ins, then each scoring-group bucket in config
// order, then each status.
func (cfg *GameYAML) MetricIDs() []string {
	ids := append([]string(nil), builtinMetricIDs...)
	if cfg == nil {
		return ids
	}
	for _, sg := range cfg.ScoringGroups {
		ids = append(ids, sg.ID)
	}
	for _, sc := range cfg.ScoringCounts {
		if sc.ScoringGroup == "" {
			ids = append(ids, sc.ID)
		}
	}
	for _, st := range cfg.Statuses {
		ids = append(ids, st.ID)
	}
	return ids
}

// MetricLabel returns the human-readable name for a metric id, falling back to the id itself for
// anything unrecognized.
func (cfg *GameYAML) MetricLabel(id string) string {
	if label, ok := builtinMetricLabels[id]; ok {
		return label
	}
	if cfg != nil {
		for _, sg := range cfg.ScoringGroups {
			if sg.ID == id && sg.DisplayName != "" {
				return sg.DisplayName
			}
		}
		for _, sc := range cfg.ScoringCounts {
			if sc.ID == id && sc.DisplayName != "" {
				return sc.DisplayName
			}
		}
		for _, st := range cfg.Statuses {
			if st.ID == id && st.DisplayName != "" {
				return st.DisplayName
			}
		}
	}
	return id
}

// ReadGameConfig parses and validates the YAML config at the given path without making it the
// active config.
func ReadGameConfig(yamlPath string) (*GameYAML, error) {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", yamlPath, err)
	}

	var cfg GameYAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", yamlPath, err)
	}

	if errs := ValidateGameYAML(&cfg); len(errs) > 0 {
		return nil, fmt.Errorf("validation errors in %s: %s", yamlPath, strings.Join(errs, "; "))
	}

	cfg.buildIndexes()
	return &cfg, nil
}

// LoadGameConfig parses, validates and activates the YAML config at the given path.
func LoadGameConfig(yamlPath string) (*GameYAML, error) {
	cfg, err := ReadGameConfig(yamlPath)
	if err != nil {
		return nil, err
	}
	SetActiveConfig(cfg)
	return cfg, nil
}

func ValidateGameYAML(cfg *GameYAML) []string {
	var validationErrors []string

	if cfg.Game.Name == "" {
		validationErrors = append(validationErrors, "game.name is required")
	}
	if cfg.Fouls.MinorFoulPoints <= 0 {
		validationErrors = append(validationErrors, "fouls.minor_foul_points must be > 0")
	}
	if cfg.Fouls.MajorFoulPoints <= 0 {
		validationErrors = append(validationErrors, "fouls.major_foul_points must be > 0")
	}

	seenIDs := make(map[string]bool)
	checkDup := func(id string, context string) {
		if id == "" {
			return
		}
		if seenIDs[id] {
			validationErrors = append(validationErrors, fmt.Sprintf("duplicate id: '%s' in %s", id, context))
		}
		seenIDs[id] = true
	}
	// Scoring groups, scoring counts and statuses all contribute metric names, so their ids may not
	// shadow a built-in metric or one of the fixed ScoreSummary/RankingFields names.
	checkReserved := func(id string, context string) {
		if reservedIDs[id] {
			validationErrors = append(
				validationErrors, fmt.Sprintf("%s: id '%s' is reserved", context, id),
			)
		}
	}

	gamePieces := make(map[string]bool)
	for i, gp := range cfg.GamePieces {
		if gp.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("game_pieces[%d]: id is required", i))
		} else if !goIdentRegexp.MatchString(gp.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("game_pieces[%d]: id '%s' must be a valid identifier", i, gp.ID))
		} else {
			checkDup(gp.ID, "game_pieces")
			gamePieces[gp.ID] = true
		}
	}

	scoringGroups := make(map[string]bool)
	for i, sg := range cfg.ScoringGroups {
		if sg.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_groups[%d]: id is required", i))
		} else if !goIdentRegexp.MatchString(sg.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_groups[%d]: id '%s' must be a valid identifier", i, sg.ID))
		} else {
			checkDup(sg.ID, "scoring_groups")
			checkReserved(sg.ID, fmt.Sprintf("scoring_groups[%d]", i))
			scoringGroups[sg.ID] = true
		}
	}

	for i, sc := range cfg.ScoringCounts {
		if sc.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d]: id is required", i))
			continue
		}
		if !goIdentRegexp.MatchString(sc.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d]: id '%s' must be a valid identifier", i, sc.ID))
			continue
		}
		checkDup(sc.ID, "scoring_counts")
		checkReserved(sc.ID, fmt.Sprintf("scoring_counts[%d]", i))

		if sc.GamePiece == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: game_piece is required", i, sc.ID))
		} else if !gamePieces[sc.GamePiece] {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: unknown game_piece '%s'", i, sc.ID, sc.GamePiece))
		}
		if sc.ScoringGroup != "" && !scoringGroups[sc.ScoringGroup] {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: unknown scoring_group '%s'", i, sc.ID, sc.ScoringGroup))
		}
		if !ValidScorers[sc.Scorer] {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: scorer must be 'near' or 'far', got '%s'", i, sc.ID, sc.Scorer))
		}

		if len(sc.Phases) == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: at least one phase is required", i, sc.ID))
			continue
		}
		seenPhases := make(map[string]bool)
		for j, ep := range sc.Phases {
			if !validElementPhases[ep.Phase] {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s.phases[%d]: unknown phase '%s'", i, sc.ID, j, ep.Phase))
				continue
			}
			if seenPhases[ep.Phase] {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: duplicate phase '%s'", i, sc.ID, ep.Phase))
			}
			seenPhases[ep.Phase] = true
			if ep.Points <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s.phases[%d]: points must be > 0", i, sc.ID, j))
			}
		}
		if seenPhases["teleop"] && seenPhases["endgame"] {
			validationErrors = append(validationErrors, fmt.Sprintf("scoring_counts[%d].%s: cannot be scored in both teleop and endgame", i, sc.ID))
		}
	}

	for i, st := range cfg.Statuses {
		if st.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d]: id is required", i))
			continue
		}
		if !goIdentRegexp.MatchString(st.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d]: id '%s' must be a valid identifier", i, st.ID))
			continue
		}
		checkDup(st.ID, "statuses")
		checkReserved(st.ID, fmt.Sprintf("statuses[%d]", i))
		if !ValidScorers[st.Scorer] {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: scorer must be 'near' or 'far', got '%s'", i, st.ID, st.Scorer))
		}

		if len(st.Phases) != 1 {
			validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: exactly one phase is required", i, st.ID))
		} else if !validStatusPhases[st.Phases[0].Phase] {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"statuses[%d].%s.phases[0]: unknown phase '%s'; only auto and endgame are supported for statuses",
					i,
					st.ID,
					st.Phases[0].Phase,
				),
			)
		}

		if len(st.Values) > 0 {
			if len(st.Values) < 2 {
				validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: enum status requires at least 2 values", i, st.ID))
			}
			if st.Values[0].Points != 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: baseline enum value '%s' must have points: 0", i, st.ID, st.Values[0].ID))
			}
			seenVals := make(map[string]bool)
			for j, val := range st.Values {
				if val.ID == "" {
					validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s values[%d]: id is required", i, st.ID, j))
				} else if seenVals[val.ID] {
					validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: duplicate value id '%s'", i, st.ID, val.ID))
				}
				seenVals[val.ID] = true
			}
		} else if len(st.Phases) == 1 {
			if st.Phases[0].Points <= 0 {
				validationErrors = append(validationErrors, fmt.Sprintf("statuses[%d].%s: points must be > 0 for bool status", i, st.ID))
			}
		}
	}

	for i, rp := range cfg.RPs {
		if rp.ID == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_points[%d]: id is required", i))
		} else if !goIdentRegexp.MatchString(rp.ID) {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_points[%d]: id '%s' must be a valid identifier", i, rp.ID))
		} else {
			checkDup(rp.ID, "ranking_points")
		}
		if rp.LogicFunc == "" || !goIdentRegexp.MatchString(rp.LogicFunc) {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_points[%d].logic_func: '%s' is not a valid identifier", i, rp.LogicFunc))
		}
	}

	validMetrics := make(map[string]bool)
	for _, id := range cfg.MetricIDs() {
		validMetrics[id] = true
	}

	seenRankingTb := make(map[string]bool)
	for i, tb := range cfg.RankingTiebreakers {
		if !validMetrics[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_tiebreakers[%d]: unknown metric '%s'", i, tb.Metric))
		} else if seenRankingTb[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("ranking_tiebreakers[%d]: duplicate metric '%s'", i, tb.Metric))
		}
		seenRankingTb[tb.Metric] = true
	}

	seenPlayoffTb := make(map[string]bool)
	for i, tb := range cfg.PlayoffTiebreakers {
		if !validMetrics[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("playoff_tiebreakers[%d]: unknown metric '%s'", i, tb.Metric))
		} else if seenPlayoffTb[tb.Metric] {
			validationErrors = append(validationErrors, fmt.Sprintf("playoff_tiebreakers[%d]: duplicate metric '%s'", i, tb.Metric))
		}
		seenPlayoffTb[tb.Metric] = true
	}

	return validationErrors
}
