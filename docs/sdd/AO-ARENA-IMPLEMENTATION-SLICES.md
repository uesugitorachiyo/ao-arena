# AO Arena Implementation Slices

These slices are written for a junior engineer implementing the future
`../ao-arena` Go repository. Each slice is independently testable.

## Slice 01: Go CLI Foundation

Create:

- `go.mod`
- `cmd/arena/main.go`
- `internal/cli/cli.go`
- `internal/cli/cli_test.go`
- `README.md`

Commands:

```sh
go test ./...
go vet ./...
go run ./cmd/arena --help
```

Acceptance:

- `arena --help` lists `suite`, `competitor`, `run`, `evidence`, `score`,
  `compare`, `report`, `gate`, and `safety`.
- `go test ./...` passes.

## Slice 02: Contract Schemas And Fixtures

Create:

- `docs/contracts/*.schema.json`
- `examples/suites/valid/ao-arena-v0.1.json`
- `examples/suites/invalid/missing-task-id.json`
- all valid and invalid fixtures named in `AO-ARENA-CONTRACTS.md`

Commands:

```sh
go test ./...
python3 -m json.tool examples/suites/valid/ao-arena-v0.1.json
```

Acceptance:

- all valid fixtures parse as JSON;
- each invalid fixture is covered by a Go test expecting validation failure.

## Slice 03: Suite And Competitor Validation

Implement:

- `arena suite validate --suite <path>`
- `arena competitor validate --competitor <path>`

Acceptance:

- canonical suite validates;
- missing task ID fails;
- live competitor without opt-in fails.

## Slice 04: Fixture Runner

Implement:

- `arena run fixture --suite <path> --competitor <path> --out <dir>`

Acceptance:

- creates deterministic attempt records for all eight tasks;
- writes evidence bundle JSON per attempt;
- never invokes live providers;
- fails if output path would be durable public docs.

## Slice 05: Evidence Bundle And Safety Scan

Implement:

- `arena evidence validate --bundle <path>`
- `arena safety scan --path <path> --out <json>`

Acceptance:

- valid evidence bundle passes;
- local absolute path fixture fails;
- secret-like fixture fails without printing the secret value;
- forbidden action fixture fails.

## Slice 06: Scoring Engine

Implement:

- `arena score --attempt <path> --scorecard <path> --out <path>`

Acceptance:

- worked bare Codex example scores 38;
- worked AO orchestration example scores 95;
- score over 100 fixture fails;
- penalties stack deterministically.

## Slice 07: Comparison Report

Implement:

- `arena compare --suite <path> --fixture-mode --out <path>`
- `arena report render --report <json> --out <markdown>`

Acceptance:

- comparison report includes baseline, challenger, per-task scores, aggregate
  scores, winner, safety status, and evidence paths;
- markdown report is derived from JSON;
- missing baseline fixture fails.

## Slice 08: Promotion Gate

Implement:

- `arena gate promotion --report <json> --out <json>`

Acceptance:

- pass fixture requires challenger score >= 85 and at least five points above
  baseline;
- unsafe report fails promotion;
- tie report fails promotion.

## Slice 09: AO Foundry Evidence Import

Implement fixture-mode import helpers for:

- Foundry GoalRun readiness JSON;
- Foundry active-stack readiness JSON;
- Forge packet summary JSON;
- AO2 run summary JSON;
- Covenant policy decision JSON.

Acceptance:

- imports are evidence inputs only;
- imports do not imply approval;
- missing source file fails closed.

## Slice 10: Public Demo Pack

Create:

- `docs/demo/BARE-CODEX-VS-AO.md`
- `examples/reports/valid/ao-arena-v0.1.comparison.md`

Acceptance:

- demo runs from clean clone in fixture mode;
- no live credentials required;
- report explains why AO orchestration won or lost.

## Final Verification

```sh
go test ./...
go vet ./...
arena suite validate --suite examples/suites/valid/ao-arena-v0.1.json
arena compare --suite examples/suites/valid/ao-arena-v0.1.json --fixture-mode --out tmp/arena-report.json
arena report render --report tmp/arena-report.json --out tmp/arena-report.md
arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json
arena safety scan --path examples --out tmp/arena-safety-scan.json
git diff --check
```
