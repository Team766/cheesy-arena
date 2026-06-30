//go:build custom

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Hand-written, config-agnostic framework tests. Config-specific correctness — point math, the
// tiebreak cascade, ranking sort, and the Score mutators — is owned by the generated_*_test.go
// files (regenerated per custom_game.yaml). Everything here uses only always-present fields.

func TestAddScoreSummary(t *testing.T) {
	fields := &RankingFields{}
	own := &ScoreSummary{Score: 15, BonusRankingPoints: 1}
	opponent := &ScoreSummary{Score: 10}

	fields.AddScoreSummary(own, opponent, false)

	assert.Equal(t, 1, fields.Played)
	assert.Equal(t, 1, fields.Wins)
	assert.Equal(t, 4, fields.RankingPoints) // 3 for the win + 1 bonus RP
}

func TestGetAllRulesCustom(t *testing.T) {
	rules := GetAllRules()
	assert.NotEmpty(t, rules)
	for id, rule := range rules {
		assert.NotNil(t, GetRuleById(id))
		assert.Equal(t, rule, GetRuleById(id))
	}
}

func TestFoulPointValueCustom(t *testing.T) {
	fMajor := Foul{IsMajor: true}
	fMinor := Foul{IsMajor: false}

	assert.Equal(t, MajorFoulPoints, fMajor.PointValue())
	assert.Equal(t, MinorFoulPoints, fMinor.PointValue())
}
