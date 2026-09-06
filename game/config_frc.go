//go:build !custom

package game

// applyGameConfigConstants is a no-op in the standard FRC build; the game name and foul point
// values are compile-time constants there. The custom build overrides this to read them from the
// loaded YAML config.
func applyGameConfigConstants(*GameYAML) {}
