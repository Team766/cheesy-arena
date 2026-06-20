// Package generate holds the //go:generate directive for the custom game code generator.
// This package is never imported by the FMS binary.
package generate

//go:generate go run ../cmd/generate -f ../game/game.yaml
