// Code generators for the web UI surfaces — scoring panel, referee panel, and audience display
// (HTML + JS) — produced by executing the committed templates/static custom_*.tmpl sources.
// Companion to codegen_go.go (server Go) and codegen_tests.go.

package main

import "path/filepath"

// DisplayBucket collapses ScoringCounts elements into one audience-display entry, per the
// fallback chain: DisplayGroup (if set) -> GamePiece (if set) -> the element's own id/display
// name. DisplayGroup and GamePiece are independent FK namespaces (an id could coincidentally
// collide between the two lists), so lookups are keyed separately to avoid cross-namespace
// false matches; ID/DisplayName on the result are always the resolved, presentable values.
type DisplayBucket struct {
	ID          string
	DisplayName string
	ElementIDs  []string
}

func buildDisplayGroups(yamlData *GameYAML) []DisplayBucket {
	var buckets []DisplayBucket
	seen := make(map[string]int)

	for _, sc := range yamlData.ScoringCounts {
		var lookupKey, groupID, displayName string
		switch {
		case sc.DisplayGroup != "":
			lookupKey = "dg:" + sc.DisplayGroup
			groupID = sc.DisplayGroup
			displayName = sc.DisplayGroup
			for _, dg := range yamlData.DisplayGroups {
				if dg.ID == sc.DisplayGroup {
					displayName = dg.DisplayName
				}
			}
		case sc.GamePiece != "":
			lookupKey = "gp:" + sc.GamePiece
			groupID = sc.GamePiece
			displayName = sc.GamePiece
			for _, gp := range yamlData.GamePieces {
				if gp.ID == sc.GamePiece {
					displayName = gp.DisplayName
				}
			}
		default:
			lookupKey = "el:" + sc.ID
			groupID = sc.ID
			displayName = sc.DisplayName
		}

		if idx, ok := seen[lookupKey]; ok {
			buckets[idx].ElementIDs = append(buckets[idx].ElementIDs, sc.ID)
		} else {
			seen[lookupKey] = len(buckets)
			buckets = append(buckets, DisplayBucket{ID: groupID, DisplayName: displayName, ElementIDs: []string{sc.ID}})
		}
	}

	return buckets
}

var phaseSectionTitle = map[string]string{"auto": "Auto", "teleop": "Teleop", "endgame": "Endgame"}

func generateScoringPanelTemplate(yamlData *GameYAML, templatesDir string) error {
	return renderWebTemplate(
		filepath.Join(templatesDir, "custom_scoring_panel.html.tmpl"),
		filepath.Join(templatesDir, "generated_scoring_panel.html"),
		buildTemplateData(yamlData),
	)
}

func generateScoringPanelJS(yamlData *GameYAML, staticJsDir string) error {
	return renderWebTemplate(
		filepath.Join(staticJsDir, "custom_scoring_panel.js.tmpl"),
		filepath.Join(staticJsDir, "generated_scoring_panel.js"),
		buildTemplateData(yamlData),
	)
}

func generateAudienceDisplayTemplate(yamlData *GameYAML, templatesDir string) error {
	return renderWebTemplate(
		filepath.Join(templatesDir, "custom_audience_display.html.tmpl"),
		filepath.Join(templatesDir, "generated_audience_display.html"),
		buildTemplateData(yamlData),
	)
}

func generateAudienceDisplayJS(yamlData *GameYAML, staticJsDir string) error {
	return renderWebTemplate(
		filepath.Join(staticJsDir, "custom_audience_display.js.tmpl"),
		filepath.Join(staticJsDir, "generated_audience_display.js"),
		buildTemplateData(yamlData),
	)
}

func generateRefereePanelTemplate(yamlData *GameYAML, templatesDir string) error {
	return renderWebTemplate(
		filepath.Join(templatesDir, "custom_referee_panel.html.tmpl"),
		filepath.Join(templatesDir, "generated_referee_panel.html"),
		buildTemplateData(yamlData),
	)
}

func generateRefereePanelJS(yamlData *GameYAML, staticJsDir string) error {
	return renderWebTemplate(
		filepath.Join(staticJsDir, "custom_referee_panel.js.tmpl"),
		filepath.Join(staticJsDir, "generated_referee_panel.js"),
		buildTemplateData(yamlData),
	)
}
