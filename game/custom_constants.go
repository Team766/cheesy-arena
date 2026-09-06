//go:build custom

package game

import (
	"log"
	"strings"
)

const CustomGameMode = true
const UseShifts = false

// CustomGameName, MinorFoulPoints and MajorFoulPoints are populated from the loaded game config.
// They keep non-zero defaults so that code paths which run before a config is loaded (unit tests
// that never call LoadTestConfig, for instance) still behave sanely.
var CustomGameName = "Custom Game"

var MinorFoulPoints = 5
var MajorFoulPoints = 15

// applyGameConfigConstants copies the config values that are exposed as package-level variables. It
// is called by SetActiveConfig; passing nil restores the built-in defaults.
func applyGameConfigConstants(cfg *GameYAML) {
	if cfg == nil {
		CustomGameName = "Custom Game"
		MinorFoulPoints = 5
		MajorFoulPoints = 15
		return
	}
	CustomGameName = cfg.Game.Name
	MinorFoulPoints = cfg.Fouls.MinorFoulPoints
	MajorFoulPoints = cfg.Fouls.MajorFoulPoints
}

// MustLoadGameConfig loads, validates and activates the game config at the given path, additionally
// checking that every ranking point's logic_func is registered. Any failure is fatal; a custom-game
// build must never run a match against a config it could not fully understand.
func MustLoadGameConfig(yamlPath string) *GameYAML {
	cfg, err := LoadGameConfig(yamlPath)
	if err != nil {
		log.Fatalf("custom game config: %v", err)
	}
	if errs := ValidateHandlers(cfg); len(errs) > 0 {
		log.Fatalf("custom game config: %s", strings.Join(errs, "; "))
	}
	return cfg
}
