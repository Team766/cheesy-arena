//go:build !custom

package model

import "github.com/Team254/cheesy-arena/game"

func initDefaultThresholds(es *EventSettings) {
	es.EnergizedBonusThreshold = game.EnergizedBonusThreshold
	es.SuperchargedBonusThreshold = game.SuperchargedBonusThreshold
	es.TraversalBonusThreshold = game.TraversalBonusThreshold
}
