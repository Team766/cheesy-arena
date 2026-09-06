//go:build !custom

package field

import (
	"testing"
)

// setupTestGameConfig is a no-op in the standard FRC build, which has no YAML game configuration.
func setupTestGameConfig(*testing.T) {}
