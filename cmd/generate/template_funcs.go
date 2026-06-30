// Helpers callable from the generate-time templates ([[ ]] funcs). They render structured
// view-model data (plain []string) into JS array literals, so the view model never has to carry
// pre-formatted JavaScript and the templates stay self-documenting. Emitting the leading "[" from
// a function (rather than as literal template text) also sidesteps the [[ ]]-delimiter collision a
// literal "[" immediately before a "[[" action would otherwise cause.

package main

import (
	"fmt"
	"strings"
	"text/template"
)

var genTemplateFuncs = template.FuncMap{
	"jsStrings":    jsStrings,
	"jsScoreArray": jsScoreArray,
	"add":          func(a, b int) int { return a + b },
	// first returns the first n elements — used to build each ranking-Less tier's prior fields.
	"first": func(xs []string, n int) []string { return xs[:n] },
	// camel is the one naming primitive: an id -> its Go/JS CamelCase identity, e.g.
	// "structure1_level1" -> "Structure1Level1". The generated field/method/const names are this
	// plus a literal suffix in the template ("Count", "Statuses", "Points", "PointsVal", "Status"),
	// so the naming convention lives at the call site instead of as precomputed view-model fields.
	"camel": toCamelCase,
	// phasePrefix is a phase's field-name prefix, e.g. "auto" -> "Auto" (so "Auto"+camel(id)+"Count"
	// is the Score field, "Phase"+prefix is the Phase enum constant, prefix+"Points" the phase total).
	"phasePrefix": func(phase string) string { return phaseFieldPrefix[phase] },
	// displayNames projects an enum status's values to their display names, for jsStrings.
	"displayNames": func(vs []ValueView) []string {
		names := make([]string, len(vs))
		for i, v := range vs {
			names[i] = v.DisplayName
		}
		return names
	},
}

// jsStrings renders display names as a JS array of quoted string literals: ["None", "Full"].
func jsStrings(xs []string) string {
	quoted := make([]string, len(xs))
	for i, x := range xs {
		quoted[i] = fmt.Sprintf("%q", x)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// jsScoreArray renders Score field names as a JS array of accessors: [score.AutoHullCount, score.AutoDeckCount].
func jsScoreArray(fields []string) string {
	exprs := make([]string, len(fields))
	for i, f := range fields {
		exprs[i] = "score." + f
	}
	return "[" + strings.Join(exprs, ", ") + "]"
}
