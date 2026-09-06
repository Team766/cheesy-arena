//go:build custom

package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidateGameConfig(t *testing.T) {
	yamlPath := filepath.Join("..", "game", "custom_game.yaml")
	cfg, err := LoadGameConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load custom_game.yaml: %v", err)
	}

	if cfg.Game.Name != "My Custom Game" {
		t.Errorf("Expected game name 'My Custom Game', got '%s'", cfg.Game.Name)
	}

	if cfg.Fouls.MinorFoulPoints != 5 || cfg.Fouls.MajorFoulPoints != 15 {
		t.Errorf("Unexpected foul points: minor=%d, major=%d", cfg.Fouls.MinorFoulPoints, cfg.Fouls.MajorFoulPoints)
	}

	errs := ValidateHandlers(cfg)
	if len(errs) > 0 {
		t.Errorf("Handler validation errors: %v", errs)
	}
}

func TestScoreMutatorsAndAccessors(t *testing.T) {
	score := new(Score)

	if !score.AdjustCount("structure1_level1", PhaseAuto, 2) {
		t.Errorf("Expected AdjustCount to return true for new count")
	}
	if score.GetCount("structure1_level1", PhaseAuto) != 2 {
		t.Errorf("Expected count 2, got %d", score.GetCount("structure1_level1", PhaseAuto))
	}

	score.AdjustCount("structure1_level1", PhaseAuto, -5)
	if score.GetCount("structure1_level1", PhaseAuto) != 0 {
		t.Errorf("Expected clamped count 0, got %d", score.GetCount("structure1_level1", PhaseAuto))
	}

	score.SetBoolStatus("park", 0, true)
	score.SetBoolStatus("park", 2, true)
	if score.CountBoolStatus("park") != 2 {
		t.Errorf("Expected 2 parked robots, got %d", score.CountBoolStatus("park"))
	}
	if !score.AnyBoolStatus("park") {
		t.Errorf("Expected AnyBoolStatus('park') to be true")
	}

	score.SetEnumStatus("muster", 1, 2)
	if score.CountEnumStatus("muster", 1) != 1 {
		t.Errorf("Expected 1 robot mustered at least partial, got %d", score.CountEnumStatus("muster", 1))
	}
}

func TestScoreSummarize(t *testing.T) {
	yamlPath := filepath.Join("..", "game", "custom_game.yaml")
	cfg, err := LoadGameConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	SetActiveConfig(cfg)

	score := new(Score)
	score.AdjustCount("structure1_level1", PhaseAuto, 3)
	score.AdjustCount("structure1_level2", PhaseTeleop, 2)
	score.SetBoolStatus("park", 0, true)
	score.SetBoolStatus("park", 1, true)

	summary := score.Summarize(nil)

	if summary.AutoPoints != 9 {
		t.Errorf("Expected AutoPoints 9, got %d", summary.AutoPoints)
	}
	if summary.TeleopPoints != 6 {
		t.Errorf("Expected TeleopPoints 6, got %d", summary.TeleopPoints)
	}
	if summary.EndgamePoints != 4 {
		t.Errorf("Expected EndgamePoints 4, got %d", summary.EndgamePoints)
	}
	if summary.MatchPoints != 19 {
		t.Errorf("Expected MatchPoints 19, got %d", summary.MatchPoints)
	}

	if !summary.RPs["auton_rp"] {
		t.Errorf("Expected auton_rp to be achieved")
	}
	if !summary.RPs["endgame_rp"] {
		t.Errorf("Expected endgame_rp to be achieved")
	}
	if summary.BonusRankingPoints != 2 {
		t.Errorf("Expected 2 bonus RPs, got %d", summary.BonusRankingPoints)
	}
}

func TestPlayoffTiebreakers(t *testing.T) {
	tmpConfig := `
game:
  name: "Test"
fouls:
  minor_foul_points: 5
  major_foul_points: 15
playoff_tiebreakers:
  - metric: auto_points
`
	tmpFile := filepath.Join(t.TempDir(), "test_game.yaml")
	if err := os.WriteFile(tmpFile, []byte(tmpConfig), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadGameConfig(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	SetActiveConfig(cfg)

	red := &ScoreSummary{Score: 50, AutoPoints: 20}
	blue := &ScoreSummary{Score: 50, AutoPoints: 10}

	status, _ := DetermineMatchStatus(red, blue, true)
	if status != RedWonMatch {
		t.Errorf("Expected Red to win via auto_points tiebreaker, got %v", status)
	}
}
