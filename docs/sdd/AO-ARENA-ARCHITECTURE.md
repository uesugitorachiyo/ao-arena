# AO Arena Architecture

## System Role

AO Arena is a local-first Go CLI. It turns benchmark suite definitions into
attempt records, evidence bundles, scorecards, comparison reports, and promotion
gate results.

The v0.1 architecture has two runner modes:

- `fixture` mode, deterministic and public-safe, used by default;
- `live` mode, explicitly out of scope for default v0.1 execution and gated by
  future operator opt-in.

## Future Repository Layout

```text
ao-arena/
  cmd/arena/main.go
  internal/cli/
  internal/benchmark/
  internal/runner/
  internal/evidence/
  internal/scoring/
  internal/report/
  internal/safety/
  docs/contracts/
  docs/design/
  examples/suites/
  examples/competitors/
  examples/attempts/
  examples/evidence/
  examples/scorecards/
  examples/reports/
  scripts/
```

## CLI Surface

| Command | Purpose |
| --- | --- |
| `arena suite validate --suite <path>` | Validate a benchmark suite and all referenced task IDs. |
| `arena competitor validate --competitor <path>` | Validate runner identity and trust boundary. |
| `arena run fixture --suite <path> --competitor <path> --out <dir>` | Materialize deterministic attempt evidence. |
| `arena evidence validate --bundle <path>` | Validate evidence bundle structure and public safety. |
| `arena score --attempt <path> --scorecard <path> --out <path>` | Produce deterministic attempt score. |
| `arena compare --suite <path> --fixture-mode --out <path>` | Compare bare Codex and AO orchestration attempts. |
| `arena report render --report <json> --out <markdown>` | Render operator-readable comparison report. |
| `arena gate promotion --report <json> --out <json>` | Emit pass/fail promotion gate result. |
| `arena safety scan --path <path> --out <json>` | Detect secrets, local paths, and forbidden actions. |

## Package Boundaries

| Package | Responsibility |
| --- | --- |
| `internal/benchmark` | Load suites, tasks, categories, prompts, expected evidence, and stop conditions. |
| `internal/runner` | Implement fixture runner and future live runner interface. |
| `internal/evidence` | Write attempt records, artifact inventories, command logs, and evidence bundles. |
| `internal/scoring` | Compute scorecards with deterministic weights and penalties. |
| `internal/report` | Produce comparison JSON and Markdown reports. |
| `internal/safety` | Scan artifacts for secrets, local paths, and forbidden actions. |
| `internal/cli` | Parse commands, route package calls, and format terminal output. |

## Data Flow

```text
benchmark suite
  -> competitor profile
  -> fixture runner
  -> attempt record
  -> evidence bundle
  -> safety scan
  -> scorecard
  -> comparison report
  -> promotion gate
```

Every output is JSON first. Markdown reports are derived views. The scoring
engine must be able to recompute a score from saved attempt and evidence files
without rerunning a benchmark.

## Storage Layout

Default output path:

```text
tmp/arena/
  attempts/<suite_id>/<competitor_id>/<task_id>/attempt.json
  attempts/<suite_id>/<competitor_id>/<task_id>/evidence-bundle.json
  scorecards/<suite_id>/<competitor_id>/<task_id>.scorecard.json
  reports/<suite_id>.comparison.json
  reports/<suite_id>.comparison.md
  gates/<suite_id>.promotion-gate.json
```

Durable examples live under `examples/`. Scratch output lives under `tmp/`.

## Integration Boundaries

AO Arena consumes AO artifacts but does not own their authority:

- Foundry GoalRun and readiness outputs are evidence inputs.
- Forge packets are evidence inputs.
- AO2 run summaries are evidence inputs.
- Covenant policy decisions are evidence inputs.
- Command summaries are operator-facing evidence inputs.
- Control-plane readback is observer evidence only.

AO Arena never treats observer readback as approval and never mutates sibling
repositories in fixture mode.
