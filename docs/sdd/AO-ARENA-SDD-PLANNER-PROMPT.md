# AO Arena SDD Planner Prompt

Use AO2 `sdd-planner` to create a junior-engineer-ready implementation plan for
the AO Arena. AO Arena is not scaffolded yet. This planning target is
docs-only and exists so the planner can reason over stable file paths before
implementation begins.

## Objective

Produce an `ao2.sdd-plan.v1` plan that fills the gaps in the AO Arena
until the plan is implementation-ready for a junior engineer.

AO Arena is the benchmark and scoring layer for recursive system improvement in
the AO orchestration framework. Its first public product must compare
bare-bones stock Codex prompts against AO orchestration workflows using
deterministic, replayable, public-safe evidence.

## Target Files

Use only these existing target paths:

- `README.md`
- `docs/sdd/AO-ARENA-PRD.md`
- `docs/sdd/AO-ARENA-ARCHITECTURE.md`
- `docs/sdd/AO-ARENA-CONTRACTS.md`
- `docs/sdd/AO-ARENA-SCORING.md`
- `docs/sdd/AO-ARENA-BENCHMARK-SUITE.md`
- `docs/sdd/AO-ARENA-SAFETY.md`
- `docs/sdd/AO-ARENA-IMPLEMENTATION-SLICES.md`
- `docs/sdd/AO-ARENA-ACCEPTANCE-GATES.md`
- `docs/sdd/AO-ARENA-SDD-HANDOFF.md`

Do not reference implementation files that do not exist yet.

## Required SDD Content

The resulting SDD work must define:

1. Product requirements:
   - users;
   - jobs to be done;
   - v0.1 scope;
   - non-goals;
   - public-safe default behavior;
   - explicit relationship to AO Foundry, AO Forge, AO2, AO Covenant, AO
     Command, and ao2-control-plane.

2. Architecture:
   - future Go CLI layout;
   - command surface;
   - package boundaries;
   - data flow from benchmark spec to report;
   - fixture mode versus live mode;
   - public artifact layout;
   - integration boundaries.

3. Contracts:
   - benchmark suite schema;
   - benchmark task schema;
   - competitor profile schema;
   - attempt record schema;
   - evidence bundle schema;
   - scorecard schema;
   - comparison report schema;
   - promotion gate result schema;
   - valid fixture names;
   - invalid fixture names;
   - exact validation commands expected after implementation.

4. Scoring:
   - exact 100-point scoring formula;
   - category weights;
   - penalties;
   - tie-breakers;
   - minimum pass thresholds;
   - deterministic worked scoring examples.

5. Benchmark suite:
   - exactly eight v0.1 benchmark tasks;
   - two single-repo implementation tasks;
   - two multi-file refactor tasks;
   - one bug fix with regression test;
   - one production-readiness hardening task;
   - one cross-repo orchestration task;
   - one overnight autonomous advancement task;
   - for each task: bare Codex prompt, AO orchestration prompt, expected
     evidence, stop condition, likely failure modes, scoring expectations.

6. Safety:
   - public-safety scan requirements;
   - secret patterns;
   - local absolute path detection;
   - forbidden actions: push, tag, release, upload, deploy, credential storage,
     sibling repo mutation;
   - live-run opt-in;
   - fail-closed behavior.

7. Implementation slices:
   - exact future files to create in the eventual `ao-arena` repo;
   - exact tests per slice;
   - exact commands per slice;
   - expected RED/GREEN behavior;
   - what should fail;
   - what should pass;
   - no vague instructions.

8. Acceptance gates:
   - the SDD pack is 100/100 only when a junior engineer can implement without
     guessing;
   - every contract has valid and invalid fixture requirements;
   - every benchmark has expected evidence;
   - scoring is deterministic;
   - safety checks are explicit;
   - implementation slices have executable verification commands.

9. Handoff:
   - final prompt for AO Foundry or AO Forge to scaffold and implement AO Arena;
   - no live-provider requirement for v0.1;
   - clear stop condition when the implementation is production-ready.

## Required Plan Shape

Create a dependency-ordered plan with 8 to 12 steps. Each step must point to one
or more target files above and must have concrete acceptance criteria. The plan
must not implement AO Arena code. It must only make the SDD pack complete enough
for implementation.

## Exit Criteria

The generated plan must include these verification commands:

- `ao2 sdd validate --plan <generated-plan.json>`
- `python3 -m json.tool <future-fixture>.json`
- `git diff --check`

The plan must require later implementation verification commands:

- `go test ./...`
- `go vet ./...`
- `arena suite validate --suite examples/suites/ao-arena-v0.1.json`
- `arena compare --suite examples/suites/ao-arena-v0.1.json --fixture-mode --out tmp/arena-report.json`
- `arena report render --report tmp/arena-report.json --out tmp/arena-report.md`
- `arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json`

## Non-Goals

- Do not scaffold the `ao-arena` implementation repository in this SDD plan.
- Do not run live providers.
- Do not call external APIs.
- Do not mutate sibling repositories.
- Do not publish, push, tag, upload, deploy, or release.
- Do not store private handoff material in public docs.
