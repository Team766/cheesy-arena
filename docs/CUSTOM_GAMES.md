# Custom Game Mode

Cheesy Arena normally implements one FRC game per season, hard-coded into the binary. Custom game
mode is an alternative build (Go build tag `custom`) that runs the same field-management system for
a game you define in one YAML file. Scoring, the scoring and referee panels, the audience display,
match review, rankings and reports all read that file at startup; nothing is generated.

Use it for off-season events, non-FRC competitions, demos, or any event whose game is not the
current FRC season's. For a complete worked example with screenshots, see
[EXAMPLE_CUSTOM_GAME.md](EXAMPLE_CUSTOM_GAME.md).

## What you edit

| File | What it holds |
|------|---------------|
| `game/custom_game.yaml` | The game: name, foul values, game pieces, scoring elements and their points, per-robot statuses, bonus ranking points, tiebreakers. |
| `game/custom_scoring_logic.go` | One short Go function per bonus ranking point. This is the only Go a game designer writes. |
| `game/custom_rules.go` | The referee's rule list (the foul dropdown on the referee panel). |
| `static/img/`, `static/audio/`, `static/css/custom_*.css` | Optional branding, sounds and styling. See [Assets you can swap](#assets-you-can-swap). |

Everything else is generic. No YAML id or display name appears anywhere else in the codebase.

## Quick start

```bash
# 1. Define the game
vim game/custom_game.yaml

# 2. Write the ranking-point functions the YAML names
vim game/custom_scoring_logic.go

# 3. (Optional) edit the rule list
vim game/custom_rules.go

# 4. Build the custom binary and run it from the repository root
go build -tags custom .
./cheesy-arena -dev
```

Open `http://localhost:8080`. `-dev` is for testing on a machine whose IP is not `10.0.100.5`.
The stock FRC binary is still `go build .`; the two builds share one codebase and one database
format, but a database created by one should not be used with the other.

### The config path and startup validation

The custom build reads `game/custom_game.yaml` relative to the working directory, or the file
named by `-game-config`:

```bash
./cheesy-arena -dev -game-config game/examples/high_seas_havoc.yaml
```

The config is a hard requirement: there is no fallback game. If the file is missing, does not
parse, fails validation, or names a `logic_func` that `game/custom_scoring_logic.go` never
registered, the server prints the reason and exits before the arena is created:

```
custom game config: error reading game/custom_game.yaml: open game/custom_game.yaml: no such file or directory
custom game config: validation errors in game/custom_game.yaml: scoring_groups[0]: id 'score' is reserved
custom game config: ranking_points[0].logic_func "ComputeCoopRP" is not registered (registered: ComputeAdventurerRP, ComputeExplorerRP, ComputeSummitRP)
```

Validation checks ids are unique, well-formed identifiers; every `game_piece` and
`scoring_group` reference exists; phases are `auto`, `teleop` or `endgame` with positive points;
a count is not scored in both teleop and endgame; a status has exactly one phase, `auto` or
`endgame`; a multi-value status has at least two values whose first is worth 0; `scorer` is
`near` or `far`; and every tiebreaker metric exists. The ids `score`, `match_points`,
`foul_points`, `ranking_points`, `auto_points`, `teleop_points`, `endgame_points` and
`total_points` are reserved.

## Defining the game

The authoritative schema is `game/config.go`. Every section below is shown with the values from
the shipped `game/custom_game.yaml`.

### `game` and `fouls`

```yaml
game:
  name: "Gem Quest"

fouls:
  minor_foul_points: 5    # awarded to the opponent per minor foul
  major_foul_points: 15   # awarded to the opponent per major foul
```

The name appears on the settings page. Foul values feed the referee panel and the score summary.

### `game_pieces`

The physical objects robots score. Each scoring count must name one. This is identity only; it
does not group anything.

```yaml
game_pieces:
  - id: gem
    display_name: "Gem"
```

### `scoring_groups`

Rollup buckets. Counts tagged with a group have their points summed into it. The group's
`display_name` labels the live counter on the audience display, a row of the post-match
breakdown, and a report column when used as a tiebreaker. A count with no group is its own
bucket under its own id.

```yaml
scoring_groups:
  - id: shelf
    display_name: "Shelf"
```

### `scoring_counts`

Countable scoring actions. Each lists the phases it can be scored in with a point value per
phase, so a piece can be worth more in auto than in teleop.

```yaml
scoring_counts:
  - id: shelf_upper
    display_name: "Upper Shelf"  # label on the scoring panel and edit page; keep it short
    game_piece: gem
    scoring_group: shelf         # optional
    scorer: near                 # optional: near | far, see "Scoring panels" below
    phases:
      - phase: auto
        points: 4
      - phase: teleop
        points: 2
```

A count may be scored in `auto` plus one of `teleop` or `endgame`, not both of the latter.

### `statuses`

Per-robot achievements, tracked for each of the three robots on an alliance. A status has exactly
one phase, `auto` or `endgame`. Without `values` it is a yes/no toggle worth `points` per robot.
With `values` it is a multi-state selector with points per state; the first value is the default
and must be worth 0.

```yaml
statuses:
  - id: bridge
    display_name: "Bridge"
    phases:
      - phase: auto
        points: 3

  - id: ascent
    display_name: "Ascent"
    scorer: far
    phases:
      - phase: endgame
    values:
      - id: none
        display_name: "None"
        points: 0
      - id: ledge
        display_name: "Ledge"
        points: 5
      - id: summit
        display_name: "Summit"
        points: 10

  - id: camp
    display_name: "Camp"
    scorer: far
    phases:
      - phase: endgame
        points: 2
```

Each status is its own bucket: its total appears on the audience breakdown under its
`display_name` and can be used as a tiebreaker metric.

### `ranking_points`

Bonus ranking points. Each names a Go function; see [Ranking-point logic](#ranking-point-logic).

```yaml
ranking_points:
  - id: adventurer_rp
    display_name: "Adventurer"
    logic_func: "ComputeAdventurerRP"
```

A win is worth 3 ranking points and a tie 1, plus one per achieved bonus, matching FRC.

### `ranking_tiebreakers` and `playoff_tiebreakers`

Ordered metric lists. `ranking_tiebreakers` sort the qualification rankings after ranking points.
`playoff_tiebreakers` break a tied playoff match; opponent major fouls are always checked first.
A metric is one of the built-ins `auto_points`, `teleop_points`, `endgame_points`,
`total_points`, a scoring-group id, or a status id.

```yaml
ranking_tiebreakers:
  - metric: total_points
  - metric: chest
  - metric: auto_points

playoff_tiebreakers:
  - metric: auto_points
  - metric: shelf
  - metric: total_points
```

Rankings reports and the rankings display show one column per ranking tiebreaker, labelled by
display name (here Total Points, Chest and Auto Points; the display abbreviates the built-ins to
Total and Auto). The audience display shows the playoff tiebreak reason as `TIEBREAK: <label>`.

## Ranking-point logic

Each `ranking_points` entry names a function in `game/custom_scoring_logic.go` (package `game`,
`//go:build custom`). Register it in `init()` under the exact `logic_func` name:

```go
func init() {
    RegisterLogicFunc("ComputeAdventurerRP", ComputeAdventurerRP)
}

// Achieved when the alliance scored at least 30 points worth of gems on the shelf and in the chest.
func ComputeAdventurerRP(score, opponentScore *Score, summary *ScoreSummary) bool {
    return summary.GroupPoints["shelf"]+summary.GroupPoints["chest"] >= 30
}
```

The function receives the alliance's own score, the opponent's score, and the alliance's
computed summary. Useful accessors:

| On `Score` | Meaning |
|-----------|---------|
| `GetCount(id, phase)` | Count scored for a scoring count in `PhaseAuto`, `PhaseTeleop` or `PhaseEndgame`. |
| `CountBoolStatus(id)`, `AnyBoolStatus(id)` | How many robots have a yes/no status; whether any does. |
| `CountEnumStatus(id, valueIndex)`, `AnyEnumStatus(id, valueIndex)` | Robots at or above a value index of a multi-value status (0 is the default). |
| `HasRankingPointFoul(ruleNumbers...)` | Whether this alliance committed a ranking-point foul under one of the rules. Call it on `opponentScore` to award an RP for opponent fouls. |
| `Fouls` | The raw foul list. |

| On `ScoreSummary` | Meaning |
|------------------|---------|
| `AutoPoints`, `TeleopPoints`, `EndgamePoints`, `MatchPoints` | Phase totals and their sum, before fouls. |
| `GroupPoints[groupId]` | Points in a scoring group (or an ungrouped count's own id). |
| `StatusPoints[statusId]` | Points from a status across the alliance. |
| `FoulPoints`, `NumOpponentMajorFouls` | Points received from opponent fouls; opponent major foul count. |

A `logic_func` that is not registered is a startup error listing the registered names.

## Rules

`game/custom_rules.go` holds the rule list the referee panel offers when assigning a foul. Each
rule has an id, a rule number, whether it is major, whether it is a ranking-point foul, and a
description. The shipped file carries the 2026 FRC rules as a starting point; replace them with
your game's. A foul is only counted by `HasRankingPointFoul` when its rule has `IsRankingPoint`
set, so keep that flag in sync with the rule numbers your ranking-point logic checks.

## Running an event

### Scoring panels

| URL | Shows |
|-----|-------|
| `/panels/scoring/red`, `/panels/scoring/blue` | Every element for that alliance. |
| `/panels/scoring/red_near`, `/panels/scoring/red_far` (and `blue_…`) | Only the elements whose `scorer` matches, plus elements with no `scorer`. |

Set `scorer: near` or `scorer: far` on counts and statuses to split an alliance's scoring between
two people; leave it off to show an element on every panel. Elements are grouped by phase.
Counts have `-`/`+` buttons; yes/no statuses are one toggle per robot under its team number;
multi-value statuses cycle through their values. Buttons are enabled while a match runs and until
the panel commits. Every connected panel must commit before Match Play can commit the match.

### Referee panel

`/panels/referee` shows both alliances' live counts and statuses (with ✓/✗ and value names),
per-robot card buttons, and the foul buttons. Assigning a foul creates a row where the referee
picks the team and the rule from `custom_rules.go`.

### Audience display

`/displays/audience` uses `templates/custom_audience_display.html.tmpl`. During a match the
overlay shows one live counter per scoring group (piece counts, not points) and the running
score. After the match is committed, the score screen shows the winner, one breakdown row per
scoring group and status (points), fouls, one ✓/✗ row per bonus ranking point, and total
ranking points, using the display names from the YAML.

### Match review and editing results

`/match_review` lists matches with their ranking-point symbols. The edit page renders one input
per scoring count and phase, a checkbox per robot for each yes/no status, and a dropdown per
robot for each multi-value status, plus fouls and cards. The score summary card refreshes as you
edit.

### Rankings and reports

Qualification rankings sort by ranking points, then the `ranking_tiebreakers` in order. The
rankings display and the CSV and PDF rankings reports show one column per tiebreaker.

### Settings

The settings page shows the game name at the top. Match timing (auto, pause, teleop and endgame
durations) is configured there as in the stock build; shift-related fields are hidden because
custom games have no shifts. TBA, Nexus and Companion integrations are disabled in custom mode
and their settings are cleared on save.

## Assets you can swap

None of these require a code change or a rebuild; static files are served as-is.

| Asset | Path | Notes |
|-------|------|-------|
| Game logo | `static/img/game-logo.png` | 600×600, shown as a circular badge in the audience match overlay and score screen. |
| Blinds logo | `static/img/blinds-logo.png` | 600×600, the large badge on the audience transition blinds. |
| Alliance station logo | `static/img/alliance-station-logo.png` | Alliance station and logo displays. |
| Lower-third logo | `static/img/lower-third-logo.png` | Lower-third overlay. |
| End-of-match background | `static/img/endofmatch-bg.png` | Audience score screen background. |
| Icons | `static/img/panel-icon.png`, `apple-icon.png`, `favicon.ico` | Home-screen and tab icons for the panels. |
| Match sounds | `static/audio/start.wav`, `end.wav`, `resume.wav`, `warning.wav`, `abort.wav` | Played at match start, end, resume after a pause, the endgame warning, and abort. `shift_change.wav` is unused in custom mode. |
| Other sounds | `static/audio/match_result.wav`, `field_reset.wav`, `pick_clock.wav`, `pick_clock_expired.wav` | Score posted, field reset signal, alliance-selection clock. |
| Audience styling | `static/css/custom_audience_display.css` | Logo sizing, live-counter boxes. Loaded on top of `audience_display.css`. |
| Scoring panel styling | `static/css/custom_scoring_panel.css` | Count and status controls. Loaded on top of `scoring_panel.css`. |
| Referee panel styling | `static/css/referee_panel.css` | Shared with the stock build. |
| Page structure | `templates/custom_audience_display.html.tmpl`, `custom_scoring_panel.html.tmpl`, `custom_referee_panel.html.tmpl` | Plain server templates; edit the markup, keep the element ids the JavaScript targets. |
| Event branding | Settings page | Event name, sponsor slides, lower thirds and awards are stock features and work unchanged. |

## Testing

```bash
go test -tags custom ./...   # custom build
go test ./...                # stock build, must stay green too
```

Tests derive their expectations from a fixture config (`game/testdata/fixture.yaml`), not from
the shipped `custom_game.yaml`, so editing your game cannot break the suite. One test,
`TestShippedConfigIsValid`, loads `game/custom_game.yaml` and every file in `game/examples/` and
validates them, including the ranking-point functions the shipped file names.

## Limitations

- Three robots per alliance.
- Statuses are per-robot and can only be `auto` or `endgame`.
- A scoring count can be scored in auto plus one of teleop or endgame.
- Team signs show scores but no game-specific text.
- TBA, Nexus and Companion integrations are unavailable.
