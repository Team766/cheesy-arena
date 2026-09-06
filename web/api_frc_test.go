//go:build !custom

package web

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// The standard FRC build has no YAML game config, so the endpoint that the custom panels use to
// discover the game must report that there is nothing to describe.
func TestGameConfigApiIsEmptyInTheStockBuild(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/api/game_config")
	assert.Equal(t, 404, recorder.Code)
	assert.Equal(t, "{}", recorder.Body.String())
}
