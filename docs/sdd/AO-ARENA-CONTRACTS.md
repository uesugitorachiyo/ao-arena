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
| Real-attempt manifest | `docs/contracts/arena-real-attempt-manifest-v0.1.schema.json` | `examples/real-attempts/valid/month5-ten-pair-manifest.json` | Runtime negative tests |
| Real-attempt task portfolio | `docs/contracts/arena-real-attempt-task-portfolio-v0.1.schema.json` | `examples/real-attempts/valid/month5-task-portfolio.json` | Runtime negative tests |
| Real-attempt outcome evidence | `docs/contracts/arena-real-attempt-evidence-v0.1.schema.json` | `examples/real-attempts/valid/evidence/*.json` | Runtime negative tests |
| Real-attempt comparison | `docs/contracts/arena-real-attempt-comparison-v0.1.schema.json` | Generated from the valid manifest | Runtime negative tests |
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

### Real-Attempt Comparison

Run the bounded comparator with:

```sh
arena compare real-attempts \
  --input examples/real-attempts/valid/month5-ten-pair-manifest.json \
  --out tmp/real-attempt-comparison.json
```

The manifest schema is `ao.arena.real-attempt-manifest.v0.1`. It contains
exactly ten pairs. Every pair has one `bare-codex` attempt and one
`ao-orchestration` attempt with exactly matching task IDs and lowercase
snapshot SHA-256 values. Task IDs are unique across pairs; all 20 attempt IDs
are unique. Scorecards bind to their attempt task and competitor and contain
all eight canonical score categories at their documented bounds. The supplied
score must exactly equal the category sum minus bounded positive penalties.

The manifest references a bounded relative
`ao.arena.real-attempt-task-portfolio.v0.1` sidecar and its exact SHA-256. The
portfolio contains exactly ten unique task contracts. Each contract fixes the
task ID, snapshot SHA-256, expected terminal, verifier-command SHA-256, and
authority-boundary SHA-256. Both attempts in a pair must match that
authoritative task contract, so an attempt cannot relabel its expected
terminal.

Attempt status tuples are fail closed:

- `completed` requires safety `passed` and stop condition `satisfied`;
- `failed` requires stop condition `failed` and may retain safety `passed` or
  `failed`;
- `blocked` requires safety `passed` and stop condition `blocked`.

Each attempt requires `regressions` and `limitations` arrays, including empty
arrays when there are no entries. Each array has at most 20 public-safe,
single-line entries of at most 500 Unicode code points. Identifiers,
derivations, penalty reasons, annotations, regressions, and limitations are
scanned before output for common token forms and Unix, Windows drive, and UNC
absolute paths, including colon/slash and `file:` forms. The schemas encode
bounded, trimmed, single-line text where JSON Schema can do so. Runtime secret
and absolute-path scanning is an intentionally stricter semantic superset;
exact token and path recognition is not represented as schema equivalence.

Each attempt also names a canonical relative JSON evidence file and its exact
SHA-256. The evidence file is bounded to 64 KiB and binds schema version,
attempt ID, task ID, competitor ID, snapshot SHA-256, expected terminal,
overall verification status, and a canonical source-result SHA-256. The
verification object also binds the task contract's verifier-command and
authority-boundary SHA-256 values, the verifier exit code, authority and
retention checks, and bounded public write and unsupported-claim counts. Every
evidence record requires
`authority_checked` and `evidence_retained` true and both counts zero. A passed
verification additionally requires exit zero; failed verification requires a
nonzero exit and never qualifies for eligibility. The
source result contains the submitted status, stop condition, safety status,
scorecard, regressions, and limitations. Its canonical digest is recomputed,
and every source-result field must exactly match the manifest before
`evidence_verified` can be true. Evidence paths and source-result digests are
validated but omitted from durable comparison output. These checks establish
that the supplied files are digest-bound and internally consistent; they do
not independently prove that an execution occurred or that its claims are
semantically true.

The manifest is bounded to 1 MiB; portfolio and evidence sidecars are each
bounded to 64 KiB. Runtime decoding rejects malformed UTF-8,
duplicate or unknown fields, trailing JSON values, non-regular inputs, and
links before reading. On supported Unix systems, reads use a nonblocking,
no-follow open, verify the opened descriptor is the same regular file observed
before opening, and only then clear nonblocking mode. The validated manifest
directory is opened once as an `os.Root`; portfolio and evidence Lstat, open,
and same-file checks are root-relative, preventing later ancestor swaps from
redirecting sidecar reads. All input and evidence validation completes before
the output is opened. Output requires an existing
non-link parent and refuses link ancestors and overwrite. The validated parent
is held through `os.Root`;
supported Unix systems use directory-handle-relative exclusive creation and
partial-file cleanup, so a later parent-path swap cannot redirect the write.
Other platforms use the standard library's conservative rooted implementation.
A partial explicit output is removed after a write, sync, or close failure when
removal is possible.

The deterministic `ao.arena.real-attempt-comparison.v0.1` output sorts by task
ID and includes effective totals and averages, per-task deltas, retained
statuses and scorecard scores, regressions, limitations, safety and
unsuccessful counts, expected terminals, overall verification statuses,
eligibility, winner, result, and tie breaker. It never contains a timestamp,
evidence path, source-result digest, or local absolute path.

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
