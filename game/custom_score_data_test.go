//go:build custom

package game

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

// The tests in this file derive every expectation from the ACTIVE config (game/testdata/fixture.yaml)
// rather than typing in ids from the shipped game/custom_game.yaml, so that editing the shipped game
// cannot break them.

func TestPhaseString(t *testing.T) {
	assert.Equal(t, "auto", PhaseAuto.String())
	assert.Equal(t, "teleop", PhaseTeleop.String())
	assert.Equal(t, "endgame", PhaseEndgame.String())
	assert.Equal(t, "", Phase(99).String())

	for _, name := range []string{"auto", "teleop", "endgame"} {
		phase, ok := PhaseFromString(name)
		assert.True(t, ok)
		assert.Equal(t, name, phase.String())
	}
	_, ok := PhaseFromString("halftime")
	assert.False(t, ok)
}

func TestAdjustCountForEveryConfiguredCountAndPhase(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	for _, sc := range cfg.ScoringCounts {
		for _, pp := range sc.Phases {
			phase, ok := PhaseFromString(pp.Phase)
			assert.True(t, ok)
			t.Run(
				sc.ID+"/"+pp.Phase, func(t *testing.T) {
					score := new(Score)
					assert.Equal(t, 0, score.GetCount(sc.ID, phase))

					assert.True(t, score.AdjustCount(sc.ID, phase, 2))
					assert.Equal(t, 2, score.GetCount(sc.ID, phase))

					assert.True(t, score.AdjustCount(sc.ID, phase, 1))
					assert.Equal(t, 3, score.GetCount(sc.ID, phase))

					// A no-op adjustment reports that nothing changed.
					assert.False(t, score.AdjustCount(sc.ID, phase, 0))

					// Going below zero clamps, and clamping still counts as a change.
					assert.True(t, score.AdjustCount(sc.ID, phase, -100))
					assert.Equal(t, 0, score.GetCount(sc.ID, phase))
					// Already at zero: decrementing changes nothing.
					assert.False(t, score.AdjustCount(sc.ID, phase, -1))
					assert.Equal(t, 0, score.GetCount(sc.ID, phase))

					// Phases are independent storage slots.
					for _, other := range []Phase{PhaseAuto, PhaseTeleop, PhaseEndgame} {
						if other != phase {
							assert.Equal(t, 0, score.GetCount(sc.ID, other))
						}
					}
				},
			)
		}

		// A phase the element does not declare is rejected outright.
		declared := make(map[string]bool)
		for _, pp := range sc.Phases {
			declared[pp.Phase] = true
		}
		for _, phase := range []Phase{PhaseAuto, PhaseTeleop, PhaseEndgame} {
			if declared[phase.String()] {
				continue
			}
			score := new(Score)
			assert.False(
				t, score.AdjustCount(sc.ID, phase, 1), "%s should not be scorable in %s", sc.ID, phase,
			)
			assert.Equal(t, 0, score.GetCount(sc.ID, phase))
		}
	}

	// An unknown element id is rejected in every phase.
	score := new(Score)
	for _, phase := range []Phase{PhaseAuto, PhaseTeleop, PhaseEndgame} {
		assert.False(t, score.AdjustCount("no_such_element", phase, 1))
	}
	assert.Empty(t, score.Counts)
}

func TestBoolStatusMutators(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	boolStatuses := 0
	for _, st := range cfg.Statuses {
		if len(st.Values) > 0 {
			// Enum statuses must reject the bool setter.
			score := new(Score)
			assert.False(t, score.SetBoolStatus(st.ID, 0, true))
			assert.False(t, score.GetBoolStatus(st.ID, 0))
			continue
		}
		boolStatuses++

		t.Run(
			st.ID, func(t *testing.T) {
				score := new(Score)
				assert.False(t, score.AnyBoolStatus(st.ID))
				assert.Equal(t, 0, score.CountBoolStatus(st.ID))

				assert.True(t, score.SetBoolStatus(st.ID, 0, true))
				assert.True(t, score.GetBoolStatus(st.ID, 0))
				assert.False(t, score.SetBoolStatus(st.ID, 0, true)) // no change
				assert.True(t, score.AnyBoolStatus(st.ID))
				assert.Equal(t, 1, score.CountBoolStatus(st.ID))

				assert.True(t, score.SetBoolStatus(st.ID, 2, true))
				assert.Equal(t, 2, score.CountBoolStatus(st.ID))
				assert.False(t, score.GetBoolStatus(st.ID, 1))

				assert.True(t, score.SetBoolStatus(st.ID, 0, false))
				assert.Equal(t, 1, score.CountBoolStatus(st.ID))

				// Out-of-range robot indexes are rejected, not clamped.
				for _, robotIndex := range []int{-1, 3, 100} {
					assert.False(t, score.SetBoolStatus(st.ID, robotIndex, true))
					assert.False(t, score.GetBoolStatus(st.ID, robotIndex))
				}
			},
		)
	}
	assert.NotZero(t, boolStatuses, "the fixture config should declare at least one bool status")

	// An unknown status id is rejected.
	score := new(Score)
	assert.False(t, score.SetBoolStatus("no_such_status", 0, true))
	assert.False(t, score.AnyBoolStatus("no_such_status"))
	assert.Equal(t, 0, score.CountBoolStatus("no_such_status"))
}

func TestEnumStatusMutators(t *testing.T) {
	cfg := LoadFixtureConfig(t)

	enumStatuses := 0
	for _, st := range cfg.Statuses {
		if len(st.Values) == 0 {
			// Bool statuses must reject the enum setters.
			score := new(Score)
			assert.False(t, score.SetEnumStatus(st.ID, 0, 1))
			assert.False(t, score.SetEnumStatusByID(st.ID, 0, "anything"))
			assert.False(t, score.CycleEnumStatus(st.ID, 0))
			continue
		}
		enumStatuses++
		st := st

		t.Run(
			st.ID, func(t *testing.T) {
				score := new(Score)
				assert.Equal(t, 0, score.GetEnumStatus(st.ID, 0))

				// Every declared value index is settable, by index and by value id.
				for i, value := range st.Values {
					if i == 0 {
						continue // already the default
					}
					assert.True(t, score.SetEnumStatus(st.ID, 1, i))
					assert.Equal(t, i, score.GetEnumStatus(st.ID, 1))
					assert.False(t, score.SetEnumStatus(st.ID, 1, i)) // no change

					assert.True(t, score.SetEnumStatusByID(st.ID, 2, value.ID))
					assert.Equal(t, i, score.GetEnumStatus(st.ID, 2))
				}
				assert.False(t, score.SetEnumStatusByID(st.ID, 0, "no_such_value"))

				// Out-of-range value indexes are rejected.
				assert.False(t, score.SetEnumStatus(st.ID, 0, -1))
				assert.False(t, score.SetEnumStatus(st.ID, 0, len(st.Values)))
				// Out-of-range robot indexes are rejected.
				assert.False(t, score.SetEnumStatus(st.ID, -1, 1))
				assert.False(t, score.SetEnumStatus(st.ID, 3, 1))
				assert.False(t, score.CycleEnumStatus(st.ID, 3))
				assert.Equal(t, 0, score.GetEnumStatus(st.ID, 3))

				// Cycling walks every value in order and wraps back to the baseline.
				cycled := new(Score)
				for i := 1; i <= len(st.Values); i++ {
					assert.True(t, cycled.CycleEnumStatus(st.ID, 0))
					assert.Equal(t, i%len(st.Values), cycled.GetEnumStatus(st.ID, 0))
				}

				// Any/Count over a threshold index.
				counted := new(Score)
				assert.False(t, counted.AnyEnumStatus(st.ID, 1))
				assert.Equal(t, 0, counted.CountEnumStatus(st.ID, 1))
				top := len(st.Values) - 1
				assert.True(t, counted.SetEnumStatus(st.ID, 0, top))
				assert.True(t, counted.SetEnumStatus(st.ID, 1, top))
				assert.True(t, counted.AnyEnumStatus(st.ID, 1))
				assert.Equal(t, 2, counted.CountEnumStatus(st.ID, top))
				assert.Equal(t, 0, counted.CountEnumStatus(st.ID, top+1))
			},
		)
	}
	assert.NotZero(t, enumStatuses, "the fixture config should declare at least one enum status")

	// An unknown status id is rejected everywhere.
	score := new(Score)
	assert.False(t, score.SetEnumStatus("no_such_status", 0, 1))
	assert.False(t, score.SetEnumStatusByID("no_such_status", 0, "x"))
	assert.False(t, score.CycleEnumStatus("no_such_status", 0))
	assert.False(t, score.AnyEnumStatus("no_such_status", 1))
	assert.Equal(t, 0, score.CountEnumStatus("no_such_status", 1))
	assert.Equal(t, 0, score.GetEnumStatus("no_such_status", 0))
}

// TestMutatorsRejectWithoutAConfig pins the post-review behavior: with no active config the score
// mutators refuse everything rather than silently accepting arbitrary ids. A build that failed to
// load its YAML must not be able to score a match.
func TestMutatorsRejectWithoutAConfig(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	// Deliberately clear the config for the body of this test; LoadFixtureConfig's cleanup restores
	// whatever was active before.
	SetActiveConfig(nil)

	countID := cfg.ScoringCounts[0].ID
	countPhase, _ := PhaseFromString(cfg.ScoringCounts[0].Phases[0].Phase)

	score := new(Score)
	assert.False(t, score.AdjustCount(countID, countPhase, 5))
	assert.Equal(t, 0, score.GetCount(countID, countPhase))

	for _, st := range cfg.Statuses {
		assert.False(t, score.SetBoolStatus(st.ID, 0, true))
		assert.False(t, score.SetEnumStatus(st.ID, 0, 1))
		assert.False(t, score.SetEnumStatusByID(st.ID, 0, "none"))
		assert.False(t, score.CycleEnumStatus(st.ID, 0))
	}

	// Summarizing a config-free build yields an empty summary rather than a panic.
	summary := score.Summarize(nil)
	assert.Equal(t, 0, summary.Score)
	assert.Empty(t, summary.RPs)
}

// scoreEverything returns a score with every count, bool status and enum status in the active config
// set to a non-default value, so that Clone/Equals/JSON tests exercise every field.
func scoreEverything(t *testing.T, cfg *GameYAML) *Score {
	t.Helper()
	score := new(Score)
	for i, sc := range cfg.ScoringCounts {
		for _, pp := range sc.Phases {
			phase, _ := PhaseFromString(pp.Phase)
			assert.True(t, score.AdjustCount(sc.ID, phase, i+1))
		}
	}
	for _, st := range cfg.Statuses {
		if len(st.Values) > 0 {
			assert.True(t, score.SetEnumStatus(st.ID, 1, len(st.Values)-1))
		} else {
			assert.True(t, score.SetBoolStatus(st.ID, 0, true))
		}
	}
	score.Fouls = []Foul{{FoulId: 1, IsMajor: true, TeamId: 254, RuleId: 1}}
	return score
}

func TestCloneIsIndependent(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	original := scoreEverything(t, cfg)

	clone := original.Clone()
	assert.True(t, original.Equals(clone))
	assert.True(t, clone.Equals(original))

	// Mutating the clone must not touch the original.
	sc := cfg.ScoringCounts[0]
	phase, _ := PhaseFromString(sc.Phases[0].Phase)
	before := original.GetCount(sc.ID, phase)
	assert.True(t, clone.AdjustCount(sc.ID, phase, 7))
	assert.Equal(t, before, original.GetCount(sc.ID, phase))
	assert.False(t, original.Equals(clone))

	clone2 := original.Clone()
	clone2.Fouls = append(clone2.Fouls, Foul{RuleId: 2})
	assert.Len(t, original.Fouls, 1)

	for _, st := range cfg.Statuses {
		clone3 := original.Clone()
		if len(st.Values) > 0 {
			assert.True(t, clone3.CycleEnumStatus(st.ID, 0))
		} else {
			assert.True(t, clone3.SetBoolStatus(st.ID, 2, true))
		}
		assert.False(t, original.Equals(clone3), "mutating %s on the clone changed the original", st.ID)
	}

	var nilScore *Score
	assert.Nil(t, nilScore.Clone())
}

func TestEqualsDetectsEverySingleMutation(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	base := scoreEverything(t, cfg)

	assert.True(t, base.Equals(base.Clone()))

	// Every count/phase slot.
	for _, sc := range cfg.ScoringCounts {
		for _, pp := range sc.Phases {
			phase, _ := PhaseFromString(pp.Phase)
			other := base.Clone()
			assert.True(t, other.AdjustCount(sc.ID, phase, 1))
			assert.False(t, base.Equals(other), "%s/%s", sc.ID, pp.Phase)
			assert.False(t, other.Equals(base), "%s/%s (symmetric)", sc.ID, pp.Phase)
		}
	}

	// Every status.
	for _, st := range cfg.Statuses {
		other := base.Clone()
		if len(st.Values) > 0 {
			assert.True(t, other.CycleEnumStatus(st.ID, 2))
		} else {
			assert.True(t, other.SetBoolStatus(st.ID, 1, true))
		}
		assert.False(t, base.Equals(other), st.ID)
	}

	// Fouls and the playoff DQ flag.
	withFoul := base.Clone()
	withFoul.Fouls = append(withFoul.Fouls, Foul{RuleId: 9})
	assert.False(t, base.Equals(withFoul))

	changedFoul := base.Clone()
	changedFoul.Fouls[0].TeamId = 1114
	assert.False(t, base.Equals(changedFoul))

	dq := base.Clone()
	dq.PlayoffDq = true
	assert.False(t, base.Equals(dq))

	// Nil handling.
	var nilScore *Score
	assert.False(t, base.Equals(nilScore))
	assert.False(t, nilScore.Equals(base))
	assert.True(t, nilScore.Equals(nil))

	// An empty score and a freshly initialized one are equal.
	assert.True(t, new(Score).Equals(new(Score)))
}

func TestCopyIntoMatchesCloneAndReusesStorage(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	source := scoreEverything(t, cfg)

	var dst Score
	source.CopyInto(&dst)
	assert.True(t, source.Equals(&dst))

	// The destination is independent of the source.
	sc := cfg.ScoringCounts[0]
	phase, _ := PhaseFromString(sc.Phases[0].Phase)
	before := source.GetCount(sc.ID, phase)
	assert.True(t, dst.AdjustCount(sc.ID, phase, 4))
	assert.Equal(t, before, source.GetCount(sc.ID, phase))

	// Copying again over a dirty destination clears stale keys rather than merging.
	source.CopyInto(&dst)
	assert.True(t, source.Equals(&dst))

	empty := new(Score)
	empty.CopyInto(&dst)
	assert.True(t, empty.Equals(&dst))
	assert.Empty(t, dst.Counts)
	assert.Empty(t, dst.Fouls)

	// Steady-state copies must not allocate; this is what makes the 10 ms arena tick affordable.
	source.CopyInto(&dst)
	allocs := testing.AllocsPerRun(
		100, func() {
			source.CopyInto(&dst)
		},
	)
	assert.Zero(t, allocs)
}

func TestScoreJsonRoundTrip(t *testing.T) {
	cfg := LoadFixtureConfig(t)
	original := scoreEverything(t, cfg)

	data, err := json.Marshal(original)
	assert.Nil(t, err)

	var decoded Score
	assert.Nil(t, json.Unmarshal(data, &decoded))
	assert.True(t, original.Equals(&decoded))
	assert.Equal(t, original.Fouls, decoded.Fouls)

	// UnmarshalJSON must initialize the maps even for a payload that omits them, so that the
	// decoded score is immediately usable by the accessors.
	var empty Score
	assert.Nil(t, json.Unmarshal([]byte(`{}`), &empty))
	assert.NotNil(t, empty.Counts)
	assert.NotNil(t, empty.BoolStatuses)
	assert.NotNil(t, empty.EnumStatuses)
	assert.Equal(t, 0, empty.GetCount(cfg.ScoringCounts[0].ID, PhaseAuto))
}
