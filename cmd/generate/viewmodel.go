// View model for the template-based UI generation. buildTemplateData performs every derivation
// the .tmpl files would otherwise have to do themselves — phase filtering, camelCasing, Go/JS
// field-name resolution, and display-group rollup — so the templates are pure iteration with no
// logic. This is the deliberate insulation layer (Option B): template authors reference this
// stable view model, never the raw GameYAML schema types, which change shape across milestones.
//
// The view model carries field-name *strings* (e.g. "AutoHullCount"), never runtime values: the
// actual numbers arrive over the websocket at match time, and the generated JS reads them off the
// live payload by these names. Because the same generator run, the same toCamelCase/phaseFieldPrefix,
// and the same YAML id produce both the Go struct field (codegen_score.go) and the string here, the
// two cannot drift.

package main

import "strings"

// playoffTiebreakerLabel returns the human label for a playoff-tiebreaker metric: fixed strings for
// the built-in point metrics, else the uppercased display name of the referenced element/status.
func playoffTiebreakerLabel(metric string, y *GameYAML) string {
	switch metric {
	case "auto_points":
		return "AUTO POINTS"
	case "teleop_points":
		return "TELEOP POINTS"
	case "endgame_points":
		return "ENDGAME POINTS"
	case "total_points":
		return "TOTAL POINTS"
	default:
		name := metric
		for _, sc := range y.ScoringCounts {
			if sc.ID == metric {
				name = sc.DisplayName
			}
		}
		for _, s := range y.Statuses {
			if s.ID == metric {
				name = s.DisplayName
			}
		}
		return strings.ToUpper(name)
	}
}

// ValueView is one named state of an enum status (or the implicit off/on of a bool status).
type ValueView struct {
	ID             string
	DisplayName    string
	Points         int
	ConstName      string // Go enum constant, e.g. "MusterNone" (enum statuses only)
	PointsValConst string // Go point constant, e.g. "MusterNonePointsVal" (enum statuses only)
}

// CountView is one scoring-count element scored in one phase. FieldName is the resolved Go/JS
// field on Score, e.g. "AutoHullCount".
type CountView struct {
	ID          string
	DisplayName string
	Phase       string
	Points      int
	FieldName   string // Score field, e.g. "AutoHullCount"
	PhaseConst  string // Go Phase constant, e.g. "PhaseAuto"
	PhasePrefix string // Go field-name phase prefix, e.g. "Auto"
}

// StatusView is one per-robot status. IsBool drives the UI choice: a filled/empty toggle for bool
// statuses vs a text-cycling button for enums. Values is always populated (bool expands to its
// implicit two states), so a template can also branch on len(.Values) if it prefers.
type StatusView struct {
	ID             string
	DisplayName    string
	Phase          string
	IsBool         bool
	Values         []ValueView
	CamelID        string   // "Muster"
	StatusesField  string   // "MusterStatuses" — the [3]bool / [3]enum array on Score
	PointsField    string   // "MusterPoints" — the per-status total on ScoreSummary
	ValueNames     []string // enum only: value display names, e.g. ["None", "Partial", "Full"]
	EnumType       string   // enum only: Go type name, e.g. "MusterStatus"
	GoElemType     string   // Go element type of the [3]_ array: "bool" (bool) or the EnumType (enum)
	NumValues      int      // enum only: number of states (for the cycle wrap-around)
	PointsValConst string   // bool only: Go point constant, e.g. "LeavePointsVal"
	PhasePointsVar string   // accumulation var the status feeds: "autoPoints" or "endgamePoints"
}

// PhaseView groups the counts and statuses scored in one phase. Only phases with at least one
// count or status are included, so a template can range over Phases without an emptiness check.
type PhaseView struct {
	Name     string // "auto"
	Title    string // "Auto"
	Counts   []CountView
	Statuses []StatusView
	// CountFields is this phase's count field names in declaration order, e.g.
	// ["AutoHullCount", "AutoDeckCount"] — the referee panel renders them as a JS accessor array.
	CountFields []string
}

// ElementView is one scoring-count element with all of its phases. Phases is element-major iteration
// (the order the scoring-panel JS updates counts in), as opposed to PhaseView's phase-major layout.
type ElementView struct {
	ID          string
	DisplayName string
	CamelID     string // "Structure1Level1" — for the AdjustXCount method name
	Phases      []CountView
}

// GroupView is one resolved audience-display rollup bucket. The audience display sums a group two
// different ways, so it carries two name lists:
//   - CountFields: raw Score count fields, phase-expanded (live mid-match counter).
//   - PointsFields: per-element ScoreSummary points fields (final breakdown).
type GroupView struct {
	ID           string
	DisplayName  string
	CamelID      string
	CountFields  []string // ["AutoHullCount", "TeleopHullCount", "AutoDeckCount", ...]
	PointsFields []string // ["HullPoints", "DeckPoints"]
}

// RPView is one ranking-point bonus. Field is the bool on ScoreSummary, e.g. "AutonRpRankingPoint".
type RPView struct {
	ID          string
	DisplayName string
	CamelID     string
	Field       string // "AutonRpRankingPoint"
	LogicFunc   string // hand-written func in custom_scoring_logic.go, e.g. "ComputeAutonRP"
}

// PointConst is one entry of the generated per-element point-constant block, e.g.
// "Structure1Level1AutoPointsVal = 4".
type PointConst struct {
	Name   string
	Points int
}

// TiebreakerView is one playoff-tiebreaker comparison: the ScoreSummary field to compare and the
// human label shown when it breaks the tie (e.g. Field "AutoPoints", Label "AUTO POINTS").
type TiebreakerView struct {
	Field string
	Label string
}

// TemplateData is the complete, stable contract exposed to the .tmpl files.
type TemplateData struct {
	GameName        string
	MinorFoulPoints int
	MajorFoulPoints int
	Phases          []PhaseView   // phase-major: UI sections laid out top-to-bottom
	ScoringCounts   []ElementView // element-major: count field accessors in declaration order
	DisplayGroups   []GroupView
	Statuses        []StatusView
	RankingPoints   []RPView
	// RankingTiebreakerFields are the RankingFields/ScoreSummary field names for each
	// ranking_tiebreakers metric, in order, e.g. ["MatchPoints", "AutoPoints"].
	RankingTiebreakerFields []string
	PointConstants          []PointConst     // per-element point-value constants (ScoreSummary)
	PlayoffTiebreakers      []TiebreakerView // DetermineMatchStatus tiebreak cascade
}

// metricFieldName maps a tiebreaker metric to its Go field name on RankingFields/ScoreSummary.
// Built-in point metrics have fixed names; any other metric (a scoring_counts/statuses id) becomes
// "{Camel}Points".
func metricFieldName(metric string) string {
	switch metric {
	case "auto_points":
		return "AutoPoints"
	case "teleop_points":
		return "TeleopPoints"
	case "endgame_points":
		return "EndgamePoints"
	case "total_points":
		return "MatchPoints"
	default:
		return toCamelCase(metric) + "Points"
	}
}

// phaseOrder is the canonical phase ordering used everywhere a UI is laid out top-to-bottom.
var phaseOrder = []string{"auto", "teleop", "endgame"}

// buildStatusView resolves a schema Status into its view form, expanding a bool status's implicit
// off/on states into Values so every status carries a populated Values slice.
func buildStatusView(status Status) StatusView {
	camel := toCamelCase(status.ID)
	phasePointsVar := "endgamePoints"
	if status.Phases[0].Phase == "auto" {
		phasePointsVar = "autoPoints"
	}
	sv := StatusView{
		ID:             status.ID,
		DisplayName:    status.DisplayName,
		Phase:          status.Phases[0].Phase,
		IsBool:         len(status.Values) < 2,
		CamelID:        camel,
		StatusesField:  camel + "Statuses",
		PointsField:    camel + "Points",
		PhasePointsVar: phasePointsVar,
	}
	if sv.IsBool {
		// Implicit two-value enum: off (0 points) / on (the bool status's points).
		sv.GoElemType = "bool"
		sv.PointsValConst = camel + "PointsVal"
		sv.Values = []ValueView{
			{ID: "false", DisplayName: "", Points: 0},
			{ID: "true", DisplayName: "", Points: status.Phases[0].Points},
		}
	} else {
		sv.EnumType = camel + "Status"
		sv.GoElemType = sv.EnumType
		sv.NumValues = len(status.Values)
		for _, v := range status.Values {
			sv.Values = append(sv.Values, ValueView{
				ID: v.ID, DisplayName: v.DisplayName, Points: v.Points,
				ConstName:      camel + toCamelCase(v.ID),
				PointsValConst: camel + toCamelCase(v.ID) + "PointsVal",
			})
			sv.ValueNames = append(sv.ValueNames, v.DisplayName)
		}
	}
	return sv
}

// buildTemplateData turns a validated GameYAML into the view model the templates consume.
func buildTemplateData(yamlData *GameYAML) TemplateData {
	td := TemplateData{
		GameName:        yamlData.Game.Name,
		MinorFoulPoints: yamlData.Fouls.MinorFoulPoints,
		MajorFoulPoints: yamlData.Fouls.MajorFoulPoints,
	}

	// Per-status views, in declaration order — used for the audience final breakdown and referee rows.
	statusViews := make([]StatusView, len(yamlData.Statuses))
	for i, status := range yamlData.Statuses {
		statusViews[i] = buildStatusView(status)
	}
	td.Statuses = statusViews

	// Phases, in canonical order, each pre-filtered to its counts and statuses. Empty phases dropped.
	for _, phase := range phaseOrder {
		pv := PhaseView{Name: phase, Title: phaseSectionTitle[phase]}
		for _, sc := range yamlData.ScoringCounts {
			for _, ep := range sc.Phases {
				if ep.Phase == phase {
					pv.Counts = append(pv.Counts, CountView{
						ID:          sc.ID,
						DisplayName: sc.DisplayName,
						Phase:       phase,
						Points:      ep.Points,
						FieldName:   phaseFieldPrefix[phase] + toCamelCase(sc.ID) + "Count",
						PhaseConst:  phaseConstName[phase],
						PhasePrefix: phaseFieldPrefix[phase],
					})
				}
			}
		}
		for _, sv := range statusViews {
			if sv.Phase == phase {
				pv.Statuses = append(pv.Statuses, sv)
			}
		}
		for _, c := range pv.Counts {
			pv.CountFields = append(pv.CountFields, c.FieldName)
		}
		if len(pv.Counts) > 0 || len(pv.Statuses) > 0 {
			td.Phases = append(td.Phases, pv)
		}
	}

	// Flat element-major view: each scoring count with its phases, in declaration order.
	for _, sc := range yamlData.ScoringCounts {
		ev := ElementView{ID: sc.ID, DisplayName: sc.DisplayName, CamelID: toCamelCase(sc.ID)}
		for _, ep := range sc.Phases {
			ev.Phases = append(ev.Phases, CountView{
				ID:          sc.ID,
				DisplayName: sc.DisplayName,
				Phase:       ep.Phase,
				Points:      ep.Points,
				FieldName:   phaseFieldPrefix[ep.Phase] + toCamelCase(sc.ID) + "Count",
				PhaseConst:  phaseConstName[ep.Phase],
				PhasePrefix: phaseFieldPrefix[ep.Phase],
			})
		}
		td.ScoringCounts = append(td.ScoringCounts, ev)
	}

	// Display-group rollup buckets, reusing the resolution chain in buildDisplayGroups.
	for _, bucket := range buildDisplayGroups(yamlData) {
		gv := GroupView{
			ID:          bucket.ID,
			DisplayName: bucket.DisplayName,
			CamelID:     toCamelCase(bucket.ID),
		}
		for _, elID := range bucket.ElementIDs {
			gv.PointsFields = append(gv.PointsFields, toCamelCase(elID)+"Points")
			// Find the element to expand its per-phase count fields.
			for i := range yamlData.ScoringCounts {
				if yamlData.ScoringCounts[i].ID == elID {
					camelEl := toCamelCase(elID)
					for _, ep := range yamlData.ScoringCounts[i].Phases {
						gv.CountFields = append(gv.CountFields, phaseFieldPrefix[ep.Phase]+camelEl+"Count")
					}
				}
			}
		}
		td.DisplayGroups = append(td.DisplayGroups, gv)
	}

	// Ranking tiebreaker field names, in order.
	for _, tb := range yamlData.RankingTiebreakers {
		td.RankingTiebreakerFields = append(td.RankingTiebreakerFields, metricFieldName(tb.Metric))
	}

	// Per-element point-value constants (ScoreSummary), counts first then statuses, in the same
	// order the generated const block declares them.
	for _, ev := range td.ScoringCounts {
		for _, c := range ev.Phases {
			td.PointConstants = append(td.PointConstants, PointConst{Name: ev.CamelID + c.PhasePrefix + "PointsVal", Points: c.Points})
		}
	}
	for _, sv := range td.Statuses {
		if sv.IsBool {
			td.PointConstants = append(td.PointConstants, PointConst{Name: sv.PointsValConst, Points: sv.Values[1].Points})
		} else {
			for _, v := range sv.Values {
				td.PointConstants = append(td.PointConstants, PointConst{Name: v.PointsValConst, Points: v.Points})
			}
		}
	}

	// Playoff tiebreaker cascade.
	for _, tb := range yamlData.PlayoffTiebreakers {
		td.PlayoffTiebreakers = append(td.PlayoffTiebreakers, TiebreakerView{
			Field: metricFieldName(tb.Metric),
			Label: "TIEBREAK: " + playoffTiebreakerLabel(tb.Metric, yamlData),
		})
	}

	// Ranking points.
	for _, rp := range yamlData.RPs {
		camel := toCamelCase(rp.ID)
		td.RankingPoints = append(td.RankingPoints, RPView{
			ID:          rp.ID,
			DisplayName: rp.DisplayName,
			CamelID:     camel,
			Field:       camel + "RankingPoint",
			LogicFunc:   rp.LogicFunc,
		})
	}

	return td
}
