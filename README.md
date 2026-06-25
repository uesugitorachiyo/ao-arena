# AO Arena

AO Arena is the deterministic benchmark and scoring layer for recursive system
improvement in the AO orchestration framework. The v0.1 product is a local-first
Go CLI that compares bare Codex prompts against AO orchestration prompts using
fixture-mode evidence, deterministic scoring, comparison reports, public-safety
scans, and promotion gates.

Fixture mode is the only default v0.1 execution path. AO Arena does not run
live providers, mutate sibling repositories, push, tag, release, upload, deploy,
or store credentials.

## Run

```sh
go test ./...
go vet ./...
go run ./cmd/arena --help
```

To run the product gate commands from a clean checkout:

```sh
go build -o tmp/bin/arena ./cmd/arena
PATH="$PWD/tmp/bin:$PATH" arena suite validate --suite examples/suites/valid/ao-arena-v0.1.json
PATH="$PWD/tmp/bin:$PATH" arena competitor validate --competitor examples/competitors/valid/bare-codex.json
PATH="$PWD/tmp/bin:$PATH" arena competitor validate --competitor examples/competitors/valid/ao-orchestration.json
PATH="$PWD/tmp/bin:$PATH" arena compare --suite examples/suites/valid/ao-arena-v0.1.json --fixture-mode --out tmp/arena-report.json
PATH="$PWD/tmp/bin:$PATH" arena report render --report tmp/arena-report.json --out tmp/arena-report.md
PATH="$PWD/tmp/bin:$PATH" arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json
PATH="$PWD/tmp/bin:$PATH" arena safety scan --path examples --out tmp/arena-safety-scan.json
git diff --check
```

## SDD Files

| File | Purpose |
| --- | --- |
| `docs/sdd/AO-ARENA-PRD.md` | Product requirements, users, scope, non-goals, success metrics. |
| `docs/sdd/AO-ARENA-ARCHITECTURE.md` | Future CLI, packages, data flow, storage layout, integrations. |
| `docs/sdd/AO-ARENA-CONTRACTS.md` | JSON contracts, fixture names, validation rules. |
| `docs/sdd/AO-ARENA-SCORING.md` | Exact scoring formula, penalties, tie rules, examples. |
| `docs/sdd/AO-ARENA-BENCHMARK-SUITE.md` | First eight benchmark tasks and comparison prompts. |
| `docs/sdd/AO-ARENA-SAFETY.md` | Public-safety, forbidden actions, live-run opt-in, fail-closed rules. |
| `docs/sdd/AO-ARENA-IMPLEMENTATION-SLICES.md` | Junior-engineer-ready implementation slices. |
| `docs/sdd/AO-ARENA-ACCEPTANCE-GATES.md` | 100/100 plan and product readiness gates. |
| `docs/sdd/AO-ARENA-SDD-HANDOFF.md` | Handoff prompt for AO Foundry or AO Forge. |

## Planner Artifacts

The validated AO2 SDD plan lives at:

- `target/ao-arena-plan.json`

The planner prompt used to derive this pack lives at:

- `docs/sdd/AO-ARENA-SDD-PLANNER-PROMPT.md`

## Implementation Rule

Implementation follows the SDD slices in order and keeps every durable artifact
public-safe. Live provider mode remains blocked unless a future profile,
operator opt-in, command flag, scratch output path, and pre/post safety scans all
authorize it.
