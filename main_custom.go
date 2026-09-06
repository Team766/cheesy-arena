//go:build custom

package main

import (
	"flag"
	"github.com/Team254/cheesy-arena/game"
)

var gameConfigPath = flag.String(
	"game-config", "game/custom_game.yaml", "Path to the custom game YAML configuration file",
)

// initCustomGame loads the custom game configuration and aborts startup if it is missing, invalid,
// or references a ranking-point logic function that isn't registered. Must be called after
// flag.Parse and before the arena is constructed.
func initCustomGame() {
	game.MustLoadGameConfig(*gameConfigPath)
}
