//go:build custom

package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Custom games have no TBA, Nexus or Companion integration: saving the settings page must clear
// those fields whatever the form submitted, and the page must not show them as enabled.
func TestSetupSettingsCustomClearsIntegrations(t *testing.T) {
	web := setupTestWeb(t)

	recorder := web.getHttpResponse("/setup/settings")
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Custom Game")

	recorder = web.postHttpResponse(
		"/setup/settings",
		"name=Gem Quest Open&playoffType=single&numPlayoffAlliances=8&tbaPublishingEnabled=on&"+
			"tbaDownloadEnabled=on&tbaEventCode=2014cc&nexusEnabled=on&nexusAutoQueueEnabled=on&"+
			"companionAddress=10.0.100.5&teleopDurationSec=120&endgameDurationSec=30",
	)
	assert.Equal(t, 303, recorder.Code)
	assert.False(t, web.arena.EventSettings.TbaPublishingEnabled)
	assert.False(t, web.arena.EventSettings.TbaDownloadEnabled)
	assert.False(t, web.arena.EventSettings.NexusEnabled)
	assert.False(t, web.arena.EventSettings.NexusAutoQueueEnabled)
	assert.Equal(t, "", web.arena.EventSettings.CompanionAddress)
	assert.Equal(t, "Gem Quest Open", web.arena.EventSettings.Name)

	recorder = web.getHttpResponse("/setup/settings")
	assert.NotContains(t, recorder.Body.String(), "tbaPublishingEnabled\"  checked")
}
