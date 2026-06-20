// Package generate holds the //go:generate directive for the custom game code generator.
// This package is never imported by the FMS binary.
//
// NOTE (M1): cmd/generate does not exist yet — `go generate ./generate/` will
// fail until M2 implements it. `go build` and `go test` are unaffected.
package generate

//go:generate go run ../cmd/generate
