# AO Arena Agent Instructions

## Status And Role

AO Arena is the active deterministic benchmark and comparative-evaluation component. It validates suites and competitors, scores recorded attempts against fixed rules, compares matched results, and renders scorecards and promotion-gate evidence.

Arena evaluates supplied files; it does not launch providers, execute AO work, modify a competitor, select a winner outside the declared formula, promote a candidate, or publish a result.

## Sources Of Truth

- [docs/sdd/AO-ARENA-PRD.md](docs/sdd/AO-ARENA-PRD.md) and [docs/sdd/AO-ARENA-ARCHITECTURE.md](docs/sdd/AO-ARENA-ARCHITECTURE.md) define product scope and evaluation flow.
- [docs/sdd/AO-ARENA-SCORING.md](docs/sdd/AO-ARENA-SCORING.md) is authoritative for scoring, penalties, eligibility, and tie handling.
- [docs/sdd/AO-ARENA-CONTRACTS.md](docs/sdd/AO-ARENA-CONTRACTS.md) and [docs/sdd/AO-ARENA-BENCHMARK-SUITE.md](docs/sdd/AO-ARENA-BENCHMARK-SUITE.md) own contract and portfolio semantics.
- `docs/contracts/`, `internal/arena/`, `internal/cli/`, and their tests own implemented validation and scoring. [`.github/workflows/ci.yml`](.github/workflows/ci.yml) defines the broad gate.
- [ao-quality-gates.json](ao-quality-gates.json) declares the source-owned commit, push, and full quality commands consumed by the stack-wide quality runner.

## Ownership And Boundaries

- Keep tasks, competitors, attempt pairs, verifier outcomes, eligibility, source snapshots, expected terminals, authority boundaries, and evidence digests fixed and reproducible.
- A real-attempt comparison proves consistency of supplied digest-bound evidence, not independent semantic truth. Retain honest failed, blocked, ineligible, and tied attempts.
- Keep valid and invalid fixtures separate. Change a benchmark, rubric, baseline, expected result, or portfolio only with explicit rationale and consuming tests; never tune after seeing a result or inflate a score.
- Refuse symlinked, traversal, absolute, malformed, unbounded, overwritten, or input-equals-output paths as required by the contracts. Keep generated reports, gates, scans, and binaries under ignored `tmp/` or `target/`.
- Do not record secrets, credentials, private source, account identifiers, user-specific paths, or unredacted provider output. The current command path remains provider-free.
- Promotion, release, deployment, publication, live provider use, credentialed operation, permission changes, and repository mutation require separate authority and are not Arena capabilities.

## Working Method

- Change the smallest benchmark or scoring surface while preserving deterministic ordering, exact arithmetic, matched-pair integrity, provenance, and failure visibility.
- Add negative tests for invalid fixtures, digest drift, portfolio mismatch, ineligible attempts, unsafe paths, overwrites, and benchmark manipulation.
- Update this file in the same pull request when durable commands, architecture, ownership, or authority boundaries change.

## Verification

- Scoring and benchmark changes: `go test ./internal/arena -count=1`.
- CLI and path-safety changes: `go test ./internal/cli -count=1`.
- Format relevant Go source with `gofmt -d` over `cmd/` and `internal/`; run `go test ./... -count=1`, `go vet ./...`, and `go build -o tmp/bin/arena ./cmd/arena`.
- Run the product-gate suite, competitor, fixture comparison, real-attempt comparison, report, promotion-gate, and safety-scan commands in [README.md](README.md) when those surfaces change.
- For instruction changes run `python3 ../ao-architecture/scripts/verify_agent_instruction_layout.py --workspace-root .. --repository ao-arena`. Always run `git diff --check`.

## Evidence And Completion

- Record source heads, suite and portfolio digests, verifier commands and exits, eligibility decisions, scoring inputs, and output digests. Disclose skipped, unavailable, blocked, or failed checks.
- Completion requires focused and broad gates, green pull-request CI, clean synchronized `main`, and task-branch cleanup. Never convert a comparative result into authority it does not carry.
