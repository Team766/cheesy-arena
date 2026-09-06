//go:build custom

package game

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fixtureLogicFuncs back the two ranking points declared by game/testdata/fixture.yaml. They are
// registered only while a test has the fixture active (see LoadFixtureConfig), never at package
// init, so the shipped binary's handler registry contains exactly what custom_scoring_logic.go
// registers.
var fixtureLogicFuncs = map[string]LogicFunc{
	"FixtureAutoRp":  FixtureAutoRp,
	"FixtureClimbRp": FixtureClimbRp,
}

// FixtureAutoRp is achieved when the alliance scores at least 6 auto points.
func FixtureAutoRp(score, opponentScore *Score, summary *ScoreSummary) bool {
	return summary.AutoPoints >= 6
}

// FixtureClimbRp is achieved when at least one robot climbs (enum value index >= 1).
func FixtureClimbRp(score, opponentScore *Score, summary *ScoreSummary) bool {
	return score.AnyEnumStatus("climb", 1)
}

// FixtureConfigPath returns the absolute path of the checked-in test fixture config.
func FixtureConfigPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		return filepath.Join(filepath.Dir(thisFile), "testdata", "fixture.yaml")
	}
	return filepath.Join("testdata", "fixture.yaml")
}

// LoadFixtureConfig makes game/testdata/fixture.yaml the active game config for the duration of the
// test, restoring the previous config afterwards. Tests in any package may call it.
func LoadFixtureConfig(t testing.TB) *GameYAML {
	t.Helper()
	cfg, err := ReadGameConfig(FixtureConfigPath())
	if err != nil {
		t.Fatalf("failed to load the test fixture game config: %v", err)
	}
	registerFixtureLogicFuncs(t)
	activateForTest(t, cfg)
	return cfg
}

// registerFixtureLogicFuncs installs the fixture's ranking-point handlers for the duration of the
// test, restoring whatever was registered under those names afterwards.
func registerFixtureLogicFuncs(t testing.TB) {
	previous := make(map[string]LogicFunc, len(fixtureLogicFuncs))
	for name, fn := range fixtureLogicFuncs {
		previous[name] = Handlers[name]
		RegisterLogicFunc(name, fn)
	}
	t.Cleanup(
		func() {
			for name, fn := range previous {
				if fn == nil {
					delete(Handlers, name)
				} else {
					Handlers[name] = fn
				}
			}
		},
	)
}

// LoadTestConfig writes the given YAML text to a temporary file, loads and validates it, and makes
// it the active game config for the duration of the test. The previous config is restored on
// cleanup.
func LoadTestConfig(t testing.TB, yamlText string) *GameYAML {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test_game.yaml")
	if err := os.WriteFile(path, []byte(yamlText), 0644); err != nil {
		t.Fatalf("failed to write the temporary game config: %v", err)
	}
	cfg, err := ReadGameConfig(path)
	if err != nil {
		t.Fatalf("failed to load the temporary game config: %v", err)
	}
	activateForTest(t, cfg)
	return cfg
}

func activateForTest(t testing.TB, cfg *GameYAML) {
	previous := GetActiveConfig()
	SetActiveConfig(cfg)
	t.Cleanup(
		func() {
			SetActiveConfig(previous)
		},
	)
}

func TestScore1() *Score {
	fouls := []Foul{
		{1, true, 25, 16},
		{2, false, 1868, 13},
		{3, false, 1868, 13},
		{4, true, 25, 15},
		{5, true, 25, 15},
		{6, true, 25, 15},
		{7, true, 25, 15},
	}
	s := &Score{
		Fouls:     fouls,
		PlayoffDq: false,
	}
	s.ensureInit()
	return s
}

func TestScore2() *Score {
	s := &Score{
		Fouls:     []Foul{},
		PlayoffDq: false,
	}
	s.ensureInit()
	return s
}

// TestRanking1/TestRanking2 are shared fixtures for the custom build (api, model, and report tests).
// They deliberately set only build-independent RankingFields — RankingPoints, the win/loss record,
// and Played — and leave the configured ranking_tiebreaker columns at their zero value, so this file
// compiles for any custom_game.yaml. Tests that need specific tiebreaker values (the rankings
// report) are generated from the config and set those columns themselves.
func TestRanking1() *Ranking {
	return &Ranking{
		TeamId: 254,
		Rank:   1,
		RankingFields: RankingFields{
			RankingPoints: 20,
			Random:        0.254,
			Wins:          3,
			Losses:        2,
			Ties:          1,
			Played:        10,
		},
	}
}

func TestRanking2() *Ranking {
	return &Ranking{
		TeamId: 1114,
		Rank:   2,
		RankingFields: RankingFields{
			RankingPoints: 18,
			Random:        0.1114,
			Wins:          1,
			Losses:        3,
			Ties:          2,
			Played:        10,
		},
	}
}
