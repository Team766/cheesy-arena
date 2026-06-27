# Custom Game Mode

## What it is

Cheesy Arena normally implements one specific FRC game per season (the stock game logic lives in files like `game/frc_constants.go`, which has no build tag and is compiled by default). Custom game mode is an alternative compile target, gated behind the Go build tag `custom`, that lets a developer define a completely different competition game — typically for an off-season event, a custom/non-FRC competition, or a demo — by editing a single YAML file (`game/game.yaml`) and one small hand-written Go file (`game/custom_scoring_logic.go`), then running a code generator.

The generator (`cmd/generate`) reads `game/game.yaml` and writes out all the Go data structures (`Score`, `ScoreSummary`, `RankingFields`), the scoring logic for computing per-match point totals, and three generated front-end surfaces (scoring panel, referee panel, audience display) as HTML templates and JS. None of the generated files are checked into git — they're listed in `.gitignore` and are expected to be produced locally by running the generator before building with `-tags custom`.

When the binary is built with `-tags custom`, `game.CustomGameMode` is `true` and the web handlers in `web/scoring_panel.go`, `web/referee_panel.go`, and `web/audience_display.go` switch to serving the generated templates instead of the stock ones. Building without the tag compiles the normal stock-game binary and ignores `game.yaml` entirely.

## Quick start

From a fresh clone, to stand up a custom game locally:

```bash
# 1. Edit the game definition
vim game/game.yaml

# 2. Write the RP logic functions referenced by game.yaml (see below)
vim game/custom_scoring_logic.go

# 3. Generate the Go structs, scoring logic, and UI surfaces
go run ./cmd/generate -f game/game.yaml

# 4. Build the custom-tagged binary
go build -tags custom -o cheesy-arena-custom .

# 5. Run it in dev mode (required outside a real field network — see below)
./cheesy-arena-custom -dev
```

Then open `http://localhost:8080` in a browser.

The generator defaults its `-f` flag to `game/game.yaml`, so `go run ./cmd/generate` with no flags works for the standard layout. Re-run step 3 any time `game/game.yaml` changes, then rebuild.

## `game.yaml` reference

The authoritative schema is `cmd/generate/schema.go` (`GameYAML` and its nested types). The committed example at `game/game.yaml` is used as the illustration below.

### `game` — top-level metadata

```yaml
game:
  name: "My Custom Game"
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `Name` | `name` | string | Display name of the game. Required by validation (`game.name is required` if empty). |

### `fouls` — foul point values

```yaml
fouls:
  minor_foul_points: 2
  major_foul_points: 6
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `MinorFoulPoints` | `minor_foul_points` | int | Points awarded to the opponent per minor foul. Must be `> 0`. |
| `MajorFoulPoints` | `major_foul_points` | int | Points awarded to the opponent per major foul. Must be `> 0`. |

### `game_pieces` — piece catalog

```yaml
game_pieces:
  - id: game_piece_1
    display_name: "Game Piece 1"
  - id: game_piece_2
    display_name: "Game Piece 2"
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `ID` | `id` | string | Must be a valid Go identifier, unique across `game_pieces`. Referenced by `scoring_counts[].game_piece`. |
| `DisplayName` | `display_name` | string | Label shown in the UI. |

`game_pieces` exists purely to declare the pieces that `scoring_counts` entries can be tagged with for grouping — it does not itself generate scoring fields.

### `scoring_counts` — auto/teleop counter elements

```yaml
scoring_counts:
  - id: gp1_level1
    display_name: "Game Piece 1 (Level 1)"
    game_piece: game_piece_1
    phase: both
    points_auto: 5
    points_teleop: 3

  - id: gp1_level2
    display_name: "Game Piece 1 (Level 2)"
    game_piece: game_piece_1
    phase: both
    points_auto: 3
    points_teleop: 1

  - id: gp2
    display_name: "Game Piece 2"
    game_piece: game_piece_2
    phase: both
    points_auto: 4
    points_teleop: 2
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `ID` | `id` | string | Valid Go identifier, unique across `scoring_counts`. Becomes a counter field on `Score` (e.g. `AutoGp1Level1Count`, `TeleopGp1Level1Count`). |
| `DisplayName` | `display_name` | string | Label shown in the UI. |
| `GamePiece` | `game_piece` | string, optional | Must reference an `id` in `game_pieces` if set. Used **only** by the audience display to group elements by piece for the live ticker and final breakdown — the scoring panel and referee panel always operate per-element, never grouped. |
| `Phase` | `phase` | `"auto"` \| `"teleop"` \| `"both"` | Which match phase(s) this element can be scored in. |
| `Points` | `points` | int | Shorthand for single-phase point value (used with `phase: auto` or `phase: teleop`). |
| `PointsAuto` | `points_auto` | int | Points per occurrence during auto. Required (with `points_teleop`) when `phase: both`. |
| `PointsTeleop` | `points_teleop` | int | Points per occurrence during teleop. Required (with `points_auto`) when `phase: both`. |
| `Group` | `group` | string, optional | UI grouping hint (not tied to `game_piece`). |

Validation rules enforced by `cmd/generate/main.go`'s `validateGameYAML`: `phase` must be one of the three values; `phase: both` requires both `points_auto > 0` and `points_teleop > 0`; `phase: auto` requires `points` or `points_auto > 0`; `phase: teleop` requires `points` or `points_teleop > 0`. The generator also normalizes `points` into `points_auto`/`points_teleop` for single-phase entries before validating.

### `endgame_counts` — endgame counter elements

```yaml
endgame_counts: []
```

Same `Element` struct as `scoring_counts`, but phase is implicitly endgame (not validated against the `auto`/`teleop`/`both` enum the way `scoring_counts` is). Each entry needs `points` (or `points_auto`/`points_teleop`) `> 0`. The committed example doesn't use this section.

### `statuses` — per-robot status flags

```yaml
statuses:
  - id: leave
    display_name: "Leave"
    phase: auto
    points: 3

  - id: park
    display_name: "Park"
    phase: endgame
    points: 2
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `ID` | `id` | string | Valid Go identifier, unique across `statuses`. Becomes a `[3]bool` or `[3]{ID}Status` array field on `Score` (3 robots per alliance), e.g. `LeaveStatuses`, `ParkStatuses`. |
| `DisplayName` | `display_name` | string | Label shown in the UI. |
| `Phase` | `phase` | `"auto"` \| `"endgame"` | Only two phases are valid for statuses (no `"teleop"`/`"both"`). |
| `Points` | `points` | int | Points awarded per robot when true. Required (`> 0`) for bool statuses (no `values`). Ignored if `values` is present. |
| `Values` | `values` | list of `StatusValue`, optional | If omitted, the status is a simple bool (`[3]bool`). If present, it must have at least 2 entries and generates a typed enum (`[3]{ID}Status`) with one Go constant and point value per state. |

`StatusValue` fields: `ID` (`id`, required, unique within the status), `DisplayName` (`display_name`), `Points` (`points`, per-state point value).

### `ranking_points` — custom RP bonus conditions

```yaml
ranking_points:
  - id: auton_rp
    display_name: "Auton Bonus"
    logic_func: "ComputeAutonRP"

  - id: scoring_rp
    display_name: "Scoring Bonus"
    logic_func: "ComputeScoringRP"

  - id: endgame_rp
    display_name: "Endgame Bonus"
    logic_func: "ComputeEndgameRP"
```

| Field | YAML key | Type | Purpose |
|---|---|---|---|
| `ID` | `id` | string | Unique RP identifier. |
| `DisplayName` | `display_name` | string | Label. |
| `LogicFunc` | `logic_func` | string | Name of a Go function that must exist in `game/custom_scoring_logic.go`, matching `func(score Score, opponentScore Score) bool`. Must be a valid Go identifier. |

The generator does not verify the function actually exists in `custom_scoring_logic.go` — it only checks that the name is a syntactically valid identifier. If you reference a function you haven't written, the build will fail with an "undefined" compiler error.

### `ranking_tiebreakers` / `playoff_tiebreakers`

```yaml
ranking_tiebreakers:
  - metric: total_points
  - metric: auto_points

playoff_tiebreakers:
  - metric: auto_points
  - metric: total_points
```

Each entry is a single field, `Metric` (`metric`), which must be one of: the built-ins `auto_points`, `teleop_points`, `endgame_points`, `total_points`, or any `id` declared in `scoring_counts`, `endgame_counts`, or `statuses`.

- `ranking_tiebreakers` generates fields on `RankingFields` and drives the `Less()` comparison used for qualification rankings, applied after ranking points and matches played.
- `playoff_tiebreakers` drives the cascade of metrics checked in `DetermineMatchStatus()` to break ties in playoff matches. Opponent major foul count is always checked first, implicitly — don't list it in this section.

## `custom_scoring_logic.go` — the one file you write by hand

`game/custom_scoring_logic.go` is build-tagged `//go:build custom` and is the **only** file in this system that the generator never overwrites. Every `logic_func` referenced in `game.yaml`'s `ranking_points` section must have a matching function here, with this signature:

```go
func ComputeXRP(score Score, opponentScore Score) bool
```

The current committed example:

```go
//go:build custom

package game

func ComputeAutonRP(score Score, opponentScore Score) bool {
	return score.AutoGp1Level1Count >= 1
}

func ComputeScoringRP(score Score, opponentScore Score) bool {
	return score.TeleopGp1Level2Count >= 1
}

func ComputeEndgameRP(score Score, opponentScore Score) bool {
	return score.ParkStatuses[0] || score.ParkStatuses[1] || score.ParkStatuses[2]
}
```

The field names referenced (`AutoGp1Level1Count`, `TeleopGp1Level2Count`, `ParkStatuses`) come straight from the generated `Score` struct — run the generator first so the fields exist, then write logic against them. If you rename or remove a `scoring_counts`/`statuses` entry in `game.yaml`, you'll need to update this file by hand to match.

## What gets generated and where

Run via `go run ./cmd/generate [-f path/to/game.yaml]`. The flag defaults to `game/game.yaml`. The generator validates the YAML (see field-level rules above) and exits with a list of errors on `stderr` if anything is invalid, otherwise it writes the following files (all gitignored, all regenerated from scratch each run — never hand-edit them):

| File | Generator function | Contents |
|---|---|---|
| `game/generated_constants.go` | `generateConstants` | `CustomGameMode = true` constant and any enum constants for status `values`. |
| `game/generated_score.go` | `generateScore` | The `Score` struct: counter fields per `scoring_counts`/`endgame_counts` entry per applicable phase, status array fields per `statuses` entry, and the score-computation logic (point totals per phase, RP evaluation calling into `custom_scoring_logic.go`). |
| `game/generated_score_summary.go` | `generateScoreSummary` | The `ScoreSummary` struct used for breakdowns/totals. |
| `game/generated_ranking_fields.go` | `generateRankingFields` | The `RankingFields` struct and `Less()` tiebreaker cascade driven by `ranking_tiebreakers`. |
| `cmd/generate/generated_score_test.go` | `generateScoreTest` | Generated unit tests for the `Score` struct's computation logic. |
| (appended into `game/generated_score.go`) | `appendRPStubs` | RP evaluation wiring that calls each `ranking_points[].logic_func`. |
| `templates/generated_scoring_panel.html` | `generateScoringPanelTemplate` | Scoring panel markup: editable counters and status toggles per element, organized by phase. |
| `static/js/generated_scoring_panel.js` | `generateScoringPanelJS` | Scoring panel websocket commands for incrementing counts / toggling statuses. |
| `templates/generated_audience_display.html` | `generateAudienceDisplayTemplate` | Audience display markup: live ticker and final breakdown table, grouped by `game_piece`. |
| `static/js/generated_audience_display.js` | `generateAudienceDisplayJS` | Audience display websocket client logic. |
| `templates/generated_referee_panel.html` | `generateRefereePanelTemplate` | Referee panel markup: read-only, per-phase raw per-element counts (not grouped by piece, not point values) plus per-robot status badges and the (shared, unmodified) fouls/cards UI. |
| `static/js/generated_referee_panel.js` | `generateRefereePanelJS` | Referee panel websocket client logic. |
| `cmd/generate/generated_template_test.go` | `generateTemplateTest` | Template-rendering smoke tests. |

**Important distinction (the thing that surprises reviewers): the audience display groups scoring elements by `game_piece` for spectator simplicity (e.g. all "Game Piece 1" levels summed into one row), but the referee panel and scoring panel never do this — they always show every `scoring_counts`/`endgame_counts` entry as its own raw count, organized by phase** (e.g. a referee panel row might read "2 / 0 / 0" for Auto, meaning 2× `gp1_level1`, 0× `gp1_level2`, 0× `gp2`, in declaration order). This was a deliberate correction during development: an early pass grouped the referee panel by piece too, but referees verify discrete field actions, not derived point totals or piece-level aggregates, so the panel was reworked to show raw per-element counts per phase instead.

## Building and running

```bash
# Regenerate after any game.yaml change
go run ./cmd/generate

# Build the custom-tagged binary
go build -tags custom -o cheesy-arena-custom .

# Run — -dev is not optional for local/non-field use
./cheesy-arena-custom -dev
```

`-dev` matters because `network.DevMode` (default `false`) controls whether driver-station listeners bind to all local interfaces. Without it, the arena networking code expects to run on the actual competition field network, where the driver station only ever tries to connect to the hardcoded `network.ServerIpAddress` (`10.0.100.5`, set in `network/switch.go`). Outside that network, omitting `-dev` will cause the relevant listeners to fail to bind, and the process exits. Always pass `-dev` for local development, testing, or any off-field event setup that isn't using the standard FRC field network gear.

Once running, hit:

- `/setup/teams` — register the teams for the event.
- `/match_play` — match control / FMS operator view.
- `/panels/scoring/red` and `/panels/scoring/blue` — generated scoring panels (one per alliance).
- `/panels/referee` — generated referee panel.
- `/displays/audience` — generated audience display (intended to be chroma-keyed/overlaid on a video feed).

## Known limitations

- The referee panel's optional foul rule-assignment dropdown (in `templates/referee_panel_foul_list.html`, shared unmodified with the stock game) still lists FRC-specific rule numbers and descriptions sourced from `game.GetAllRules()`. `game.yaml` has no concept of rules yet, so custom games inherit the stock ruleset's labels in that dropdown even though the rest of the panel is fully custom-generated. Rule assignment is optional, so this doesn't block scoring, but expect it to look out of place.
- `-tags custom` is not built or tested in CI today — only exercised locally via `go build -tags custom` and `go test -tags custom`. Don't assume a green CI run has validated the custom build; run it yourself after any change touching `cmd/generate`, `game/custom_scoring_logic.go`, or `game.yaml`.
