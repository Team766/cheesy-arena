//go:build custom

package web

import (
	"fmt"
	"github.com/Team254/cheesy-arena/game"
	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

// findElement returns the opening tag of the first element of the given type having the given
// attribute, e.g. findElement(t, body, "input", `name="redCount_foo_auto"`).
func findElement(t *testing.T, body string, tag string, attribute string) string {
	pattern := regexp.MustCompile(`<` + tag + `[^>]*` + regexp.QuoteMeta(attribute) + `[^>]*>`)
	element := pattern.FindString(body)
	assert.NotEmpty(t, element, "expected a <%s> having %s", tag, attribute)
	return element
}

// findSelectElement returns the whole <select> element (including its options) having the given
// attribute.
func findSelectElement(t *testing.T, body string, attribute string) string {
	pattern := regexp.MustCompile(`(?s)<select[^>]*` + regexp.QuoteMeta(attribute) + `[^>]*>.*?</select>`)
	element := pattern.FindString(body)
	assert.NotEmpty(t, element, "expected a <select> having %s", attribute)
	return element
}

// customEditFixture picks the config elements that the edit-page tests exercise. Everything is
// derived from the active config so that the test doesn't depend on the shipped custom_game.yaml.
type customEditFixture struct {
	config       *game.GameYAML
	autoCount    *game.ScoringCount
	boolStatus   *game.Status
	enumStatus   *game.Status
	autoCountKey string
}

func setupCustomEditFixture(t *testing.T) customEditFixture {
	fixture := customEditFixture{config: game.GetActiveConfig()}
	if !assert.NotNil(t, fixture.config, "expected an active game config") {
		t.FailNow()
	}

	for i := range fixture.config.ScoringCounts {
		count := &fixture.config.ScoringCounts[i]
		for _, phase := range count.Phases {
			if phase.Phase == "auto" && fixture.autoCount == nil {
				fixture.autoCount = count
				fixture.autoCountKey = count.ID + "_auto"
			}
		}
	}
	for i := range fixture.config.Statuses {
		status := &fixture.config.Statuses[i]
		if len(status.Values) == 0 {
			if fixture.boolStatus == nil {
				fixture.boolStatus = status
			}
		} else if fixture.enumStatus == nil {
			fixture.enumStatus = status
		}
	}

	if !assert.NotNil(t, fixture.autoCount, "config needs a count scored in auto") ||
		!assert.NotNil(t, fixture.boolStatus, "config needs a bool status") ||
		!assert.NotNil(t, fixture.enumStatus, "config needs an enum status") {
		t.FailNow()
	}
	return fixture
}

// Checks that the custom score-editing form is rendered from the game config, populated with the
// stored score, and that the stock FRC form is nowhere to be seen.
func TestMatchReviewEditCustomForm(t *testing.T) {
	web := setupTestWeb(t)
	fixture := setupCustomEditFixture(t)

	match := model.Match{
		Type: model.Practice, ShortName: "P1", Red1: 101, Red2: 102, Red3: 103, Blue1: 104, Blue2: 105, Blue3: 106,
	}
	assert.Nil(t, web.arena.Database.CreateMatch(&match))

	matchResult := model.NewMatchResult()
	matchResult.MatchId = match.Id
	matchResult.MatchType = match.Type
	matchResult.PlayNumber = 1
	assert.True(t, matchResult.RedScore.AdjustCount(fixture.autoCount.ID, game.PhaseAuto, 3))
	assert.True(t, matchResult.RedScore.SetBoolStatus(fixture.boolStatus.ID, 1, true))
	lastValueIndex := len(fixture.enumStatus.Values) - 1
	assert.True(t, matchResult.RedScore.SetEnumStatus(fixture.enumStatus.ID, 2, lastValueIndex))
	assert.Nil(t, web.arena.Database.CreateMatchResult(matchResult))

	recorder := web.getHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id))
	assert.Equal(t, 200, recorder.Code)
	body := recorder.Body.String()

	// The stock FRC score-editing form must not be rendered in custom mode.
	assert.NotContains(t, body, "HubShiftCount")
	assert.NotContains(t, body, "AutoTowerStatuses")
	assert.NotContains(t, body, "EndgameTowerStatuses")

	// Every configured count is editable, for both alliances, once per phase it is scored in.
	for _, count := range fixture.config.ScoringCounts {
		for _, phase := range count.Phases {
			key := count.ID + "_" + phase.Phase
			for _, alliance := range []string{"red", "blue"} {
				input := findElement(t, body, "input", fmt.Sprintf(`name="%sCount_%s"`, alliance, key))
				assert.Contains(t, input, `type="number"`)
				assert.Contains(t, input, fmt.Sprintf(`data-custom-count="%s"`, key))
			}
		}
		assert.Contains(t, body, count.DisplayName)
	}

	// The stored red values are pre-populated, and the blue alliance's are still at their defaults.
	assert.Contains(
		t, findElement(t, body, "input", fmt.Sprintf(`name="redCount_%s"`, fixture.autoCountKey)), `value="3"`,
	)
	assert.Contains(
		t, findElement(t, body, "input", fmt.Sprintf(`name="blueCount_%s"`, fixture.autoCountKey)), `value="0"`,
	)

	// Bool statuses render one checkbox per robot, checked only where the score says so.
	for i, expectChecked := range []bool{false, true, false} {
		input := findElement(t, body, "input", fmt.Sprintf(`name="redBoolStatus_%s_%d"`, fixture.boolStatus.ID, i))
		assert.Contains(t, input, `type="checkbox"`)
		assert.Contains(t, input, fmt.Sprintf(`data-custom-bool-status="%s"`, fixture.boolStatus.ID))
		assert.Contains(t, input, fmt.Sprintf(`data-robot-index="%d"`, i))
		assert.Equal(t, expectChecked, regexp.MustCompile(`\bchecked\b`).MatchString(input))
	}
	assert.Contains(t, body, fixture.boolStatus.DisplayName)

	// Enum statuses render one select per robot, listing every configured value by display name.
	for i, expectedIndex := range []int{0, 0, lastValueIndex} {
		element := findSelectElement(t, body, fmt.Sprintf(`name="redEnumStatus_%s_%d"`, fixture.enumStatus.ID, i))
		assert.Contains(t, element, fmt.Sprintf(`data-custom-enum-status="%s"`, fixture.enumStatus.ID))
		for j, value := range fixture.enumStatus.Values {
			assert.Contains(t, element, value.DisplayName)
			if j == expectedIndex {
				assert.Contains(t, element, fmt.Sprintf(`<option value="%d" selected>`, j))
			} else {
				assert.Contains(t, element, fmt.Sprintf(`<option value="%d">`, j))
			}
		}
	}
	assert.Contains(t, body, fixture.enumStatus.DisplayName)
}

// Checks that a manually edited custom score posted back from the edit page round-trips through the
// handler into the database.
func TestMatchReviewEditCustomPost(t *testing.T) {
	web := setupTestWeb(t)
	fixture := setupCustomEditFixture(t)

	match := model.Match{
		Type: model.Practice, ShortName: "P1", Red1: 101, Red2: 102, Red3: 103, Blue1: 104, Blue2: 105, Blue3: 106,
	}
	assert.Nil(t, web.arena.Database.CreateMatch(&match))

	lastValueIndex := len(fixture.enumStatus.Values) - 1
	redScoreJson := fmt.Sprintf(
		`{"Counts":{"%s":7},"BoolStatuses":{"%s":[true,false,true]},"EnumStatuses":{"%s":[0,%d,0]},`+
			`"Fouls":[{"TeamId":104,"RuleId":0,"IsMajor":true}]}`,
		fixture.autoCountKey,
		fixture.boolStatus.ID,
		fixture.enumStatus.ID,
		lastValueIndex,
	)
	blueScoreJson := fmt.Sprintf(`{"Counts":{"%s":2}}`, fixture.autoCountKey)
	postBody := fmt.Sprintf(
		`matchResultJson={"MatchId":%d,"RedScore":%s,"BlueScore":%s,"RedCards":{"101":"yellow"},"BlueCards":{}}`,
		match.Id,
		redScoreJson,
		blueScoreJson,
	)
	recorder := web.postHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id), postBody)
	assert.Equal(t, 303, recorder.Code, recorder.Body.String())

	savedResult, err := web.arena.Database.GetMatchResultForMatch(match.Id)
	assert.Nil(t, err)
	if !assert.NotNil(t, savedResult) {
		return
	}

	assert.Equal(t, 7, savedResult.RedScore.GetCount(fixture.autoCount.ID, game.PhaseAuto))
	assert.Equal(t, [3]bool{true, false, true}, savedResult.RedScore.BoolStatuses[fixture.boolStatus.ID])
	assert.Equal(t, [3]int{0, lastValueIndex, 0}, savedResult.RedScore.EnumStatuses[fixture.enumStatus.ID])
	assert.Equal(t, 1, len(savedResult.RedScore.Fouls))
	assert.Equal(t, map[string]string{"101": "yellow"}, savedResult.RedCards)
	assert.Equal(t, 2, savedResult.BlueScore.GetCount(fixture.autoCount.ID, game.PhaseAuto))

	// The maps must have been initialized even for the alliance whose JSON omitted them.
	assert.NotNil(t, savedResult.BlueScore.BoolStatuses)
	assert.NotNil(t, savedResult.BlueScore.EnumStatuses)

	// The edited score is reflected back in the freshly rendered form.
	recorder = web.getHttpResponse(fmt.Sprintf("/match_review/%d/edit", match.Id))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(
		t,
		findElement(t, recorder.Body.String(), "input", fmt.Sprintf(`name="redCount_%s"`, fixture.autoCountKey)),
		`value="7"`,
	)
}
