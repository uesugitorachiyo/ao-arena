# AO Arena Contracts

All AO Arena contracts are JSON-first and schema-backed. Markdown reports are
rendered from JSON and must never become the source of truth.

## Contract Families

| Contract | Schema path | Valid fixture | Invalid fixture |
| --- | --- | --- | --- |
| Benchmark suite | `docs/contracts/arena-benchmark-suite-v0.1.schema.json` | `examples/suites/valid/ao-arena-v0.1.json` | `examples/suites/invalid/missing-task-id.json` |
| Benchmark task | `docs/contracts/arena-benchmark-task-v0.1.schema.json` | `examples/tasks/valid/single-repo-feature.json` | `examples/tasks/invalid/missing-stop-condition.json` |
| Competitor profile | `docs/contracts/arena-competitor-v0.1.schema.json` | `examples/competitors/valid/ao-orchestration.json` | `examples/competitors/invalid/live-without-opt-in.json` |
| Attempt record | `docs/contracts/arena-attempt-v0.1.schema.json` | `examples/attempts/valid/fixture-attempt.json` | `examples/attempts/invalid/unsafe-action.json` |
| Evidence bundle | `docs/contracts/arena-evidence-bundle-v0.1.schema.json` | `examples/evidence/valid/evidence-bundle.json` | `examples/evidence/invalid/local-absolute-path.json` |
| Scorecard | `docs/contracts/arena-scorecard-v0.1.schema.json` | `examples/scorecards/valid/scorecard.json` | `examples/scorecards/invalid/score-over-maximum.json` |
| Comparison report | `docs/contracts/arena-comparison-report-v0.1.schema.json` | `examples/reports/valid/comparison-report.json` | `examples/reports/invalid/missing-baseline.json` |
| Promotion gate | `docs/contracts/arena-promotion-gate-v0.1.schema.json` | `examples/gates/valid/promotion-pass.json` | `examples/gates/invalid/unsafe-promotion.json` |

## Required Shapes

### Benchmark Suite

```json
{
  "schema_version": "ao.arena.benchmark-suite.v0.1",
  "suite_id": "ao-arena-v0.1",
  "title": "AO Arena v0.1 Benchmark Suite",
  "mode": "fixture",
  "tasks": ["single-repo-feature-cli-health"],
  "competitors": ["bare-codex", "ao-orchestration"],
  "safety_profile": "public-safe-fixture",
  "scorecard": "arena-default-v0.1"
}
```

Validation rules:

- `tasks` must contain exactly eight task IDs for the canonical suite.
- `competitors` must include `bare-codex` and `ao-orchestration`.
- `mode` must be `fixture` unless live mode is explicitly enabled later.
- `safety_profile` must be present.

### Benchmark Task

```json
{
  "schema_version": "ao.arena.benchmark-task.v0.1",
  "task_id": "single-repo-feature-cli-health",
  "category": "single_repo_feature",
  "title": "Add CLI health command",
  "bare_codex_prompt": "Add a CLI command that prints repo health.",
  "ao_orchestration_prompt": "Use AO orchestration to plan, gate, implement, test, and produce evidence for a repo health command.",
  "expected_evidence": ["tests", "changed_files", "operator_summary"],
  "stop_condition": "tests pass and evidence bundle validates",
  "failure_modes": ["no tests", "unclear summary", "unsafe mutation"]
}
```

Validation rules:

- `category` must be one of the canonical v0.1 categories.
- both prompt fields must be non-empty.
- `expected_evidence` must be non-empty.
- `stop_condition` must be non-empty.

### Competitor Profile

```json
{
  "schema_version": "ao.arena.competitor.v0.1",
  "competitor_id": "ao-orchestration",
  "runner": "fixture",
  "trust_boundary": {
    "mutates_sibling_repos": false,
    "requires_live_provider": false,
    "stores_credentials": false
  },
  "description": "AO Foundry/Forge/AO2-style evidence-first orchestration."
}
```

Validation rules:

- fixture mode must not require live providers.
- `stores_credentials` must be false.
- live mode profiles must fail unless `operator_live_opt_in` is true.

### Attempt Record

Attempt records bind a task, competitor, runner mode, output status, and evidence
bundle digest.

Required fields:

- `schema_version`
- `attempt_id`
- `suite_id`
- `task_id`
- `competitor_id`
- `runner`
- `status`
- `started_at_utc`
- `completed_at_utc`
- `evidence_bundle`
- `safety_scan`
- `stop_condition_status`

### Evidence Bundle

Evidence bundles contain only public-safe references:

- prompt digest;
- runner mode;
- artifact inventory;
- command log summaries;
- test result summaries;
- changed-file summaries;
- safety scan result;
- operator handoff summary.

Evidence bundles must reject:

- local absolute paths and parent traversal in durable paths;
- tokens, bearer strings, API keys, private keys, cookies, and passwords;
- push, tag, release, upload, deploy, or sibling mutation commands in fixture
  mode.

### Scorecard

Scorecards contain category scores, penalties, final score, and derivation
metadata. Every score must be reproducible from attempt and evidence JSON.

### Comparison Report

Comparison reports require:

- one baseline competitor;
- one or more challenger competitors;
- per-task scores;
- aggregate scores;
- win, loss, or tie result;
- safety status;
- evidence paths;
- operator summary.

### Promotion Gate

Promotion gates pass only when:

- challenger score is higher than baseline by at least five points;
- challenger score is at least 85;
- all required safety checks pass;
- no task has missing evidence;
- no stop condition is violated.

## Validation Commands

Future implementation must support:

```sh
arena suite validate --suite examples/suites/valid/ao-arena-v0.1.json
arena competitor validate --competitor examples/competitors/valid/ao-orchestration.json
arena evidence validate --bundle examples/evidence/valid/evidence-bundle.json
arena score --attempt examples/attempts/valid/fixture-attempt.json --scorecard examples/scorecards/valid/scorecard.json --out tmp/scorecard.json
```
