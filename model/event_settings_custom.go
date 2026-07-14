//go:build custom

package model

func initDefaultThresholds(es *EventSettings) {
	// These thresholds are not used in custom games.
	es.EnergizedBonusThreshold = 0
	es.SuperchargedBonusThreshold = 0
	es.TraversalBonusThreshold = 0
}
