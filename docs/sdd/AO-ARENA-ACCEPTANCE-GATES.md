# AO Arena Acceptance Gates

## SDD Pack 100/100 Gate

The SDD pack is ready for implementation only when:

- PRD defines users, scope, non-goals, and success metrics;
- architecture defines CLI, packages, data flow, storage, and integrations;
- contracts define schemas, valid fixtures, invalid fixtures, and validation
  commands;
- scoring defines exact math, weights, penalties, thresholds, tie rules, and
  worked examples;
- benchmark suite defines exactly eight tasks with both prompt styles;
- safety defines forbidden actions, secret patterns, path scans, live opt-in,
  and fail-closed behavior;
- implementation slices define future files, tests, commands, and acceptance;
- the maintained public SDD pack and README command path drive fixture-only
  validation, deterministic scoring, report rendering, and promotion-gate
  evaluation without private prompt context.

## Product 100/100 Gate

AO Arena product implementation is production-ready when:

```sh
go test ./...
go vet ./...
arena suite validate --suite examples/suites/valid/ao-arena-v0.1.json
arena competitor validate --competitor examples/competitors/valid/bare-codex.json
arena competitor validate --competitor examples/competitors/valid/ao-orchestration.json
arena compare --suite examples/suites/valid/ao-arena-v0.1.json --fixture-mode --out tmp/arena-report.json
arena report render --report tmp/arena-report.json --out tmp/arena-report.md
arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json
arena safety scan --path examples --out tmp/arena-safety-scan.json
git diff --check
```

All commands must exit zero except intentionally invalid fixture tests, which
must fail in expected test cases.

## Public-Safety Gate

The public-safety gate passes only when:

- no secret-like strings are present;
- no durable local absolute paths are present;
- no forbidden actions are present in fixture mode;
- no private handoff or local coordination material is included;
- generated reports redact finding values.

## Competitive Gate

The AO orchestration competitor is allowed to claim a benchmark win only when:

- aggregate score beats bare Codex by at least five points;
- aggregate score is at least 85;
- safety status is `passed`;
- every task has required evidence;
- stop conditions are satisfied.

## Stop Conditions

Stop implementation work when:

- product 100/100 gate passes;
- a blocker is outside AO Arena scope;
- a required live provider path is requested before fixture mode is complete;
- a requested action would push, tag, publish, upload, deploy, or mutate sibling
  repositories without explicit operator approval.
