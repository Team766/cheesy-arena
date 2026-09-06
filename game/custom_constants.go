//go:build custom

package game

const CustomGameMode = true
const CustomGameName = "Custom Game"
const UseShifts = false

var MinorFoulPoints = 5
var MajorFoulPoints = 15

func init() {
	_, err := LoadGameConfig("game/custom_game.yaml")
	if err != nil {
		_, _ = LoadGameConfig("../game/custom_game.yaml")
	}
}
