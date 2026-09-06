//go:build !custom

package game

// CustomGameMode indicates whether a custom off-season game build is active.
// For the standard FRC build, custom game mode is disabled and the name is empty.
const CustomGameMode = false
const CustomGameName = ""

// Standard FRC game foul point values.
const MinorFoulPoints = 5
const MajorFoulPoints = 15

// UseShifts controls whether Hub shift-change sound cues are emitted.
// Standard FRC games use shifts; custom games do not.
const UseShifts = true

// applyGameConfigConstants is a no-op in the standard FRC build; the game name and foul point
// values are compile-time constants there. The custom build overrides this to read them from the
// loaded YAML config.
func applyGameConfigConstants(*GameYAML) {}
