# AO Arena

AO Arena is AO's deterministic benchmark and comparative-evaluation tool. It validates benchmark suites and competitors, scores recorded attempts, compares results, and renders reports. Use it when two execution approaches need to be measured against the same tasks and scoring rules. Its current command path evaluates supplied evidence rather than launching providers.

## How it fits in AO

- **Primary responsibility:** Benchmarking and comparative evaluation.
- **Inputs:** Benchmark suites, competitor definitions, attempts, and evidence bundles.
- **Outputs:** Scorecards, comparison reports, and evaluation results.
- **Upstream:** AO2 runs or other recorded attempts.
- **Downstream:** AO Sentinel and AO Promoter.

See the
[AO Architecture guide](https://github.com/uesugitorachiyo/ao-architecture)
and the
[AO Arena component page](https://github.com/uesugitorachiyo/ao-architecture/blob/main/components/ao-arena.md)
for the cross-repository flow.

## Build and run from source

Requires Go 1.24 or later.

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
PATH="$PWD/tmp/bin:$PATH" arena compare real-attempts --input examples/real-attempts/valid/month5-ten-pair-manifest.json --out tmp/real-attempt-comparison.json
PATH="$PWD/tmp/bin:$PATH" arena report render --report tmp/arena-report.json --out tmp/arena-report.md
PATH="$PWD/tmp/bin:$PATH" arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json
PATH="$PWD/tmp/bin:$PATH" arena safety scan --path examples --out tmp/arena-safety-scan.json
git diff --check
```

`compare real-attempts` evaluates exactly ten matched `bare-codex` and
`ao-orchestration` attempts without calling providers. The input manifest and
its digest-bound task portfolio and relative evidence files must be regular
non-link, bounded, strict JSON. The portfolio authoritatively fixes each task's
snapshot, expected terminal, verifier command, and authority boundary.
Evidence binds verifier outcomes and the complete scored source result by
digest and exact manifest equality; this verifies supplied-file consistency,
not independent semantic truth. The report retains honest failed and blocked
attempts while applying the documented eligibility rules.
The output parent must already exist, and the command refuses symlinks,
overwrites, and input/output identity. It writes only the explicit `--out`
file, with no timestamp, evidence path, or local absolute path in the report.

## SDD Files

| File | Purpose |
| --- | --- |
| `docs/sdd/AO-ARENA-PRD.md` | Product requirements, users, scope, non-goals, success metrics. |
| `docs/sdd/AO-ARENA-ARCHITECTURE.md` | CLI, packages, data flow, storage layout, integrations. |
| `docs/sdd/AO-ARENA-CONTRACTS.md` | JSON contracts, fixture names, validation rules. |
| `docs/sdd/AO-ARENA-SCORING.md` | Exact scoring formula, penalties, tie rules, examples. |
| `docs/sdd/AO-ARENA-BENCHMARK-SUITE.md` | First eight benchmark tasks and comparison prompts. |
| `docs/sdd/AO-ARENA-SAFETY.md` | Public-safety, forbidden actions, live-run opt-in, fail-closed rules. |
| `docs/sdd/AO-ARENA-IMPLEMENTATION-SLICES.md` | Junior-engineer-ready implementation slices. |
| `docs/sdd/AO-ARENA-ACCEPTANCE-GATES.md` | 100/100 plan and product readiness gates. |

## License

AO Arena is licensed under `Apache-2.0`. See `LICENSE`.
