// Code generator module for appending ranking point stubs to custom_scoring_logic.go.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func appendRPStubs(yamlData *GameYAML, destDir string) error {
	filePath := filepath.Join(destDir, "custom_scoring_logic.go")
	contentBytes, err := os.ReadFile(filePath)
	var content string
	appendedAny := false
	if err != nil {
		if os.IsNotExist(err) {
			content = `//go:build custom

package game

// Custom scoring logic for the active custom game.
// This file is human-curated and is NOT overwritten by go generate.
// Add ComputeXRP functions referenced in the custom game's config here.
`
			appendedAny = true
		} else {
			return err
		}
	} else {
		content = string(contentBytes)
	}

	var sb strings.Builder
	sb.WriteString(content)

	for _, rp := range yamlData.RPs {
		pattern := fmt.Sprintf(`func\s+%s\s*\(`, regexp.QuoteMeta(rp.LogicFunc))
		matched, _ := regexp.MatchString(pattern, content)
		if !matched {
			if !strings.HasSuffix(sb.String(), "\n\n") {
				if strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString("\n")
				} else {
					sb.WriteString("\n\n")
				}
			}
			sb.WriteString(fmt.Sprintf("func %s(score Score, opponentScore Score) bool {\n\treturn false\n}\n", rp.LogicFunc))
			appendedAny = true
		}
	}

	if appendedAny {
		return os.WriteFile(filePath, []byte(sb.String()), 0644)
	}

	return nil
}
