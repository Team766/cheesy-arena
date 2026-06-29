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
