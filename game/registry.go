//go:build custom

package game

import (
	"fmt"
	"sort"
	"strings"
)

type LogicFunc func(score, opponentScore *Score, summary *ScoreSummary) bool

var Handlers = make(map[string]LogicFunc)

func RegisterLogicFunc(name string, fn LogicFunc) {
	Handlers[name] = fn
}

// RegisteredLogicFuncs returns the names of every registered logic function, sorted.
func RegisteredLogicFuncs() []string {
	names := make([]string, 0, len(Handlers))
	for name := range Handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func ValidateHandlers(cfg *GameYAML) []string {
	var errors []string
	if cfg == nil {
		return errors
	}
	for i, rp := range cfg.RPs {
		if _, ok := Handlers[rp.LogicFunc]; !ok {
			errors = append(
				errors,
				fmt.Sprintf(
					"ranking_points[%d].logic_func %q is not registered (registered: %s)",
					i,
					rp.LogicFunc,
					strings.Join(RegisteredLogicFuncs(), ", "),
				),
			)
		}
	}
	return errors
}
