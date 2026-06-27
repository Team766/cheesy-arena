# Custom Game Mode — Design Overview

For the field-by-field `game.yaml` schema and build/run instructions, see [CUSTOM_GAMES.md](CUSTOM_GAMES.md). This doc is for reviewers: what moves where, and why, at a glance.

## The core idea

Cheesy Arena ships one game per season, hardcoded across `game/`, `web/`, and the templates. Custom mode adds a second compile target (`-tags custom`) where those same touchpoints are driven by a YAML file instead. Nothing about the live arena/match-control logic (timing, networking, PLC, alliance selection, playoffs) changes — only the *scoring schema* and the three UI surfaces that display it.

## The two build worlds, and how they're kept apart

Every file that differs between the stock and custom game is split into a `!custom` file and a `custom` file using Go build tags, e.g.:

- `game/score.go` (`!custom`) vs `game/generated_score.go` (`custom`, gitignored, produced by the generator)
- `model/event_settings_frc.go` (`!custom`) vs `model/event_settings_custom.go` (`custom`)
- `partner/tba.go` (`!custom`) vs `partner/tba_custom.go` (`custom`, no-op stub)

Look for this pattern (`grep -rl '//go:build custom' --include='*.go'`) to find every seam between the two worlds. There's no runtime branching for most of this — the linker just picks one file or the other per build.

The one place that *is* a runtime branch is the three web handlers that pick which template to serve:

```go
// web/scoring_panel.go, web/referee_panel.go, web/audience_display.go
if game.CustomGameMode {
    template = "templates/generated_scoring_panel.html"
} else {
    template = "templates/scoring_panel.html"
}
```

`game.CustomGameMode` itself is build-tag-gated (`false` under `!custom`, generated as `true` under `custom`), so the branch is statically dead in the stock binary.

## The generator (`cmd/generate`)

A standalone CLI (`go run ./cmd/generate`), never imported by the FMS binary — its only relationship to the rest of the repo is "writes files that get compiled in." Three jobs, one per file group, see `cmd/generate/codegen_*.go`:

| Reads | Writes | Generator function |
|---|---|---|
| `game.yaml` | `game/generated_{constants,score,score_summary,ranking_fields}.go` | `codegen_score.go`, `codegen_score_summary.go`, `codegen_ranking.go`, `codegen_constants.go` |
| `game.yaml` | `templates/generated_{scoring_panel,referee_panel,audience_display}.html`, `static/js/generated_*.js` | `codegen_templates.go` (1268 lines — the largest single file, worth reading closely) |
| `game.yaml` | `cmd/generate/generated_*_test.go` | `codegen_tests.go` |

`cmd/generate/schema.go` defines the YAML shape and `main.go`'s `validateGameYAML` enforces it (valid Go identifiers, required point values, etc.) before any codegen runs — bad input fails with a list of errors on stderr rather than generating broken Go.

All generator output is gitignored (`game/generated_*.go`, `templates/generated_*.html`, `static/js/generated_*.js`) — it's a build step (`go generate ./generate/`), not a commit. Reviewing a PR that touches the generator means reading the generator *source*; the actual generated Go/HTML/JS for the example `game.yaml` can be produced locally with `go run ./cmd/generate` if you want to see real output.

## The one hand-written exception: `game/custom_scoring_logic.go`

Ranking-point bonus logic (e.g. "did this alliance score ≥3 in auto") can't be expressed declaratively in YAML, so `game.yaml`'s `ranking_points[].logic_func` names a Go function that must exist in this one file, with signature `func(score Score, opponentScore Score) bool`. It's the only `custom`-tagged file the generator never overwrites — everything else under the `custom` tag in `game/` is generated.

## Scoring panel → websocket → referee/audience panel data flow

```
Scoring panel (browser)
  → generated JS sends {Id, Phase, RobotIndex, ValueId} over websocket
  → web/scoring_panel_custom.go's adjustCount/setStatus/setEnumStatus
      (generic dispatchers — the game-specific logic lives in the generated
       Score.AdjustCount/SetBoolStatus/SetEnumStatus methods they call)
  → RealtimeScoreNotifier.Notify()
  → referee panel JS + audience display JS (separate generated_*.js files,
    both listening on the same websocket) re-render
```

The dispatch handlers in `web/scoring_panel_custom.go` never change when `game.yaml` changes — they're generic plumbing. The per-element logic (which field to increment, which point value applies) lives entirely in generated `Score` methods.

## A deliberate UI asymmetry, worth knowing before reviewing the templates

- **Audience display** groups scoring elements by `game_piece` (e.g. two count fields both tagged `game_piece: hull` get summed into one "Hull" row) — spectators want a simple total, not raw element counts.
- **Scoring panel and referee panel never group** — every `scoring_counts`/`endgame_counts` entry is shown as its own raw count, organized by phase. A referee verifies discrete field actions, not derived totals.

This is implemented via `buildPieceGroups` (shared helper, `codegen_templates.go`) for the audience display only; the other two panels iterate `game.yaml`'s elements directly with no grouping step.

## Where to look for what

| Concern | Files |
|---|---|
| Build tag plumbing / "is this seam complete" | `grep -rl '//go:build custom'` and its `!custom` counterparts |
| YAML schema + validation | `cmd/generate/schema.go`, `main.go`'s `validateGameYAML` |
| Go-struct codegen correctness | `cmd/generate/codegen_{constants,score,score_summary,ranking,rules}.go` |
| UI template/JS codegen correctness | `cmd/generate/codegen_templates.go` |
| Websocket command handling | `web/scoring_panel_custom.go`, `web/referee_panel.go` |
| RP bonus logic contract | `game/custom_scoring_logic.go` |
| The "why" behind specific UI decisions | `docs/CUSTOM_GAMES.md`'s "Important distinction" section, and `.linemate/feature_custom-games/m3_walkthrough.md`'s "Design history" section |

## Known, accepted gaps

- The referee panel's optional rule-assignment dropdown still shows FRC-specific rule numbers (`game.GetAllRules()`) — `game.yaml` has no concept of rules. Cosmetic, non-blocking.
- `-tags custom` is not built/tested in CI — only locally. A green default-build CI run says nothing about the custom build.
