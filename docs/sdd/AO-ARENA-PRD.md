# AO Arena PRD

## Product Summary

AO Arena is the benchmark and scoring layer for the AO orchestration framework.
It compares bare-bones stock Codex prompts against AO-orchestrated workflows
using deterministic tasks, evidence bundles, scorecards, comparison reports, and
promotion gates.

AO Arena exists because recursive system improvement is impossible without a
scoreboard. AO Foundry, AO Forge, AO2, AO Covenant, AO Command, and
ao2-control-plane can improve only when candidate changes are measured against a
stable baseline and rejected unless they produce better verified outcomes.

## Users

| User | Job |
| --- | --- |
| AO framework maintainer | Decide whether an orchestration change beats the current baseline. |
| Release reviewer | Inspect evidence behind a benchmark result before promotion. |
| Junior engineer | Implement benchmark contracts and fixture mode from exact docs. |
| Operator | Run a public-safe comparison without live credentials. |
| Future AO Crucible loop | Consume Arena promotion gates before suggesting framework changes. |

## v0.1 Goals

1. Define benchmark suites that compare bare Codex prompts with AO
   orchestration prompts.
2. Run deterministic fixture-mode attempts without live provider credentials.
3. Collect evidence for each attempt: prompts, declared runner, command logs,
   artifact inventory, public-safety scan, expected changes, and verifier
   results.
4. Score attempts using a deterministic 100-point rubric.
5. Produce JSON and Markdown comparison reports.
6. Emit a promotion gate result that blocks candidate AO improvements unless
   they beat the baseline and pass safety gates.

## Non-Goals

- Do not run live model providers in v0.1 default paths.
- Do not push, tag, release, upload, deploy, or mutate sibling repositories.
- Do not store secrets, tokens, private prompts, private evidence, or local
  absolute paths in durable artifacts.
- Do not replace AO Foundry, AO Forge, AO2, AO Covenant, AO Command, or
  ao2-control-plane.
- Do not claim superiority from artifact volume alone; scores must require
  correctness and evidence quality.

## Success Metrics

AO Arena v0.1 is successful when:

- eight canonical benchmark tasks validate;
- fixture-mode comparison runs are reproducible;
- every score can be recomputed from saved evidence;
- safety gates block unsafe artifacts and forbidden actions;
- AO orchestration can win only by producing better verified outcomes;
- a junior engineer can implement every slice from the SDD pack without
  guessing.

## Relationship To The AO Stack

| Component | Arena relationship |
| --- | --- |
| AO Foundry | Uses Arena reports to decide whether a self-improvement candidate is worth delegating. |
| AO Forge | Can execute implementation of accepted benchmark or scoring changes under GoalRun gates. |
| AO2 | Supplies governed execution evidence for AO-style attempts in later live modes. |
| AO Covenant | Supplies safety and policy concepts for forbidden actions, approvals, and fail-closed gates. |
| AO Command | Can summarize Arena reports for operators. |
| ao2-control-plane | May store published observer readback after explicit operator promotion. |

## Production Readiness Definition

The SDD pack is implementation-ready when it specifies exact future files,
commands, schemas, fixtures, score formulas, benchmark tasks, failure cases, and
verification gates. The product is production-ready when `go test ./...`,
`go vet ./...`, suite validation, fixture comparison, report rendering, safety
scan, and promotion gate commands all pass from a clean clone.
