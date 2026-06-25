# AO Arena SDD Handoff

Use this prompt after the SDD pack is reviewed.

```text
You are implementing AO Arena v0.1 from the approved SDD pack.

Repository to create:
./ao-arena

Source SDD pack:
./ao-arena

Goal:
Build AO Arena as the deterministic benchmark and scoring layer for the AO
orchestration framework. The v0.1 product compares bare Codex prompts against
AO orchestration prompts through fixture-mode evidence, deterministic scoring,
comparison reports, safety scans, and promotion gates.

Required constraints:
- Start with fixture mode only.
- Do not run live providers.
- Do not push, tag, release, upload, deploy, or mutate sibling repositories.
- Do not store secrets or local absolute paths in durable examples.
- Implement slice by slice from AO-ARENA-IMPLEMENTATION-SLICES.md.
- After each slice, run focused tests and update evidence.
- Stop when AO-ARENA-ACCEPTANCE-GATES.md product 100/100 gate passes.

First commands:
- inspect the SDD pack;
- create the Go CLI foundation;
- add schema and fixture validation tests before implementation logic;
- run `go test ./...`, `go vet ./...`, and `git diff --check`.

Final response must include:
- slices completed;
- files changed;
- verification commands and results;
- current production-readiness score;
- remaining blocking next actions, if any.
```

## Implementation Readiness Verdict

The plan is ready to implement only when:

- `target/ao-arena-plan.json` validates with AO2 SDD validation;
- all SDD docs contain concrete requirements rather than placeholders;
- acceptance gates define exact commands;
- the handoff prompt above needs no extra context.
