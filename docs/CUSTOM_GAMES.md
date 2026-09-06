# Custom Game Mode

## Overview

Cheesy Arena normally implements one specific FRC game per season, hardcoded into the binary. Custom game mode is an alternative compile target (Go build tag `custom`) that drives the same arena/match-control system from a YAML config and runtime engine instead — scoring rules, referee/scoring/audience UIs, and rankings all read dynamically from `game/custom_game.yaml` rather than generated code.

Use it for off-season events, custom or non-FRC competitions, demos, or any event running a game that isn't the current FRC season's.

## Quick Start

```bash
# 1. Edit the game definition
vim game/custom_game.yaml

# 2. Write/register the RP logic functions referenced by custom_game.yaml
vim game/custom_scoring_logic.go

# 3. Build and run
go build -tags custom
./cheesy-arena [-dev]
```

Open `http://<host>:8080`. `-dev` is required when testing the FMS locally rather than on a machine with IP `10.0.100.5`.

Any changes to `custom_game.yaml` are validated and loaded at server startup.

## How to define a custom game

### Schema (`custom_game.yaml`)

The authoritative schema is defined in `game/config.go`.

**`game`** — top-level metadata.
```yaml
game:
  name: "My Custom Game"
```

**`fouls`** — points awarded to the opponent per foul.
```yaml
fouls:
  minor_foul_points: 5
  major_foul_points: 15
```

**`game_pieces`** — the game pieces being manipulated by alliance robots.
```yaml
game_pieces:
  - id: game_piece_1
    display_name: "Game Piece 1"
  - id: game_piece_2
    display_name: "Game Piece 2"
```

**`scoring_groups`** — rollup buckets. A `scoring_counts` entry tagged with a `scoring_group` has its live count and points summed into that bucket in the score summary and audience display.
```yaml
scoring_groups:
  - id: structure1
    display_name: "Struct 1"
  - id: structure2
    display_name: "Struct 2"
```

**`scoring_counts`** — countable scoring actions.
```yaml
scoring_counts:
  - id: structure1_level1
    display_name: "S1L1"
    game_piece: game_piece_1
    scoring_group: structure1
    phases:
      - phase: auto
        points: 3
      - phase: teleop
        points: 1
```

**`statuses`** — per-robot status flags for auto or endgame achievements.
```yaml
statuses:
  - id: leave
    display_name: "Leave"
    phases:
      - phase: auto
        points: 3
  - id: park
    display_name: "Park"
    phases:
      - phase: endgame
        points: 2
```

**`ranking_points`** — custom RP bonus conditions, backed by registered Go logic functions.
```yaml
ranking_points:
  - id: auton_rp
    display_name: "Auto Bonus"
    logic_func: "ComputeAutonRP"
```

**`ranking_tiebreakers`** / **`playoff_tiebreakers`** — metric cascades for sorting rankings and breaking playoff ties.
```yaml
ranking_tiebreakers:
  - metric: total_points
  - metric: auto_points

playoff_tiebreakers:
  - metric: auto_points
  - metric: total_points
```

### Custom RP Scoring

In `game/custom_scoring_logic.go` (`//go:build custom`), logic functions are registered in `init()`:

```go
func init() {
    game.RegisterLogicFunc("ComputeAutonRP", ComputeAutonRP)
}

func ComputeAutonRP(score, opponentScore *game.Score, summary *game.ScoreSummary) bool {
    return score.GetCount("structure1_level1", game.PhaseAuto) > 2
}
```

## Verification & Testing

Run unit tests across packages under custom mode:

```bash
go test -tags custom ./...
```
