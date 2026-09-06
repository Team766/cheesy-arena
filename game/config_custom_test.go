//go:build custom

package game

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateHandlers(t *testing.T) {
	cfg := testGameYAML()

	// The fixture names ComputeAutoRp, which no build registers.
	errs := ValidateHandlers(cfg)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0], `ranking_points[0].logic_func "ComputeAutoRp" is not registered`)
	// The message must list what *is* registered so a typo is fixable without reading the source.
	assert.Contains(t, errs[0], "registered: ")
	for _, name := range RegisteredLogicFuncs() {
		assert.Contains(t, errs[0], name)
	}

	// The fixture's handlers are not part of the shipped registry; they exist only while a test has
	// installed them.
	cfg.RPs = []RankingPoint{{ID: "auto_rp", LogicFunc: "FixtureAutoRp"}}
	assert.Len(t, ValidateHandlers(cfg), 1)
	registerFixtureLogicFuncs(t)
	assert.Empty(t, ValidateHandlers(cfg))

	// A config with no ranking points, and a nil config, both pass.
	cfg.RPs = nil
	assert.Empty(t, ValidateHandlers(cfg))
	assert.Empty(t, ValidateHandlers(nil))
}

func TestRegisteredLogicFuncsIsSorted(t *testing.T) {
	names := RegisteredLogicFuncs()
	assert.NotEmpty(t, names)
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i])
	}
	assert.Len(t, names, len(Handlers))
}

// TestShippedConfigIsValid is the runtime replacement for the old "it generated" compile-time
// guarantee: the config that actually ships must parse, validate, and name only logic functions
// that game/custom_scoring_logic.go registers.
func TestShippedConfigIsValid(t *testing.T) {
	// Tests run with the package directory as the working directory.
	gameDir := "."

	t.Run(
		"custom_game.yaml", func(t *testing.T) {
			cfg, err := ReadGameConfig(filepath.Join(gameDir, "custom_game.yaml"))
			if !assert.Nil(t, err) {
				return
			}
			assert.NotEmpty(t, cfg.Game.Name)
			assert.Empty(t, ValidateHandlers(cfg))
		},
	)

	entries, err := os.ReadDir(filepath.Join(gameDir, "examples"))
	if !assert.Nil(t, err) {
		return
	}
	assert.NotEmpty(t, entries, "expected at least one example config under game/examples")
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		t.Run(
			"examples/"+entry.Name(), func(t *testing.T) {
				cfg, err := ReadGameConfig(filepath.Join(gameDir, "examples", entry.Name()))
				if !assert.Nil(t, err) {
					return
				}
				assert.NotEmpty(t, cfg.Game.Name)
				// Handlers are deliberately NOT checked for the examples: an example is a schema
				// sample, and swapping it in is expected to come with its own
				// custom_scoring_logic.go. Only the shipped config is required to be runnable
				// against the shipped logic file.
			},
		)
	}
}

// TestShippedConfigLogicFuncsAreAllUsed guards against a logic function that no longer has a
// ranking point pointing at it, which is dead code the validator cannot see.
func TestShippedConfigLogicFuncsAreAllUsed(t *testing.T) {
	cfg, err := ReadGameConfig("custom_game.yaml")
	if !assert.Nil(t, err) {
		return
	}
	used := make(map[string]bool)
	for _, rp := range cfg.RPs {
		used[rp.LogicFunc] = true
	}
	for _, name := range RegisteredLogicFuncs() {
		if strings.HasPrefix(name, "Fixture") {
			// Registered by test_helpers_custom.go for game/testdata/fixture.yaml.
			continue
		}
		assert.True(t, used[name], "logic func %q is registered but no ranking point references it", name)
	}
}
