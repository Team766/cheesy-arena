//go:build custom

package game

import (
	"fmt"
)

type LogicFunc func(score, opponentScore *Score, summary *ScoreSummary) bool

var Handlers = make(map[string]LogicFunc)

func RegisterLogicFunc(name string, fn LogicFunc) {
	Handlers[name] = fn
}

func ValidateHandlers(cfg *GameYAML) []string {
	var errors []string
	if cfg == nil {
		return errors
	}
	for _, rp := range cfg.RPs {
		if _, ok := Handlers[rp.LogicFunc]; !ok {
			errors = append(errors, fmt.Sprintf("ranking point '%s': logic_func '%s' is not registered in Handlers", rp.ID, rp.LogicFunc))
		}
	}
	return errors
}
