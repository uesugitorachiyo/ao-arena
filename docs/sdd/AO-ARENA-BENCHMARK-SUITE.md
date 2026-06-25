# AO Arena Benchmark Suite

The v0.1 suite contains exactly eight canonical tasks. Each task compares a
bare Codex prompt with an AO orchestration prompt.

## Task 1: Single-Repo Feature, CLI Health

- `task_id`: `single-repo-feature-cli-health`
- Category: `single_repo_feature`
- Bare Codex prompt: `Add a CLI command that prints repo health.`
- AO prompt: `Use AO orchestration to plan, gate, implement, test, and produce evidence for a repo health command.`
- Expected evidence: tests, changed-file summary, command log, operator handoff.
- Stop condition: command works, tests pass, evidence validates.
- Likely bare failure: no structured evidence or missing tests.

## Task 2: Single-Repo Feature, JSON Inspect

- `task_id`: `single-repo-feature-json-inspect`
- Category: `single_repo_feature`
- Bare Codex prompt: `Add an inspect command that reads a JSON file and prints a summary.`
- AO prompt: `Plan a bounded JSON inspect feature with schema validation, tests, evidence, and stop condition.`
- Expected evidence: valid fixture, invalid fixture, parser tests, safety scan.
- Stop condition: valid JSON passes, invalid JSON fails with clear error.
- Likely bare failure: ad hoc parsing or weak fixture coverage.

## Task 3: Multi-File Refactor, Split CLI Logic

- `task_id`: `multi-file-refactor-cli-boundary`
- Category: `multi_file_refactor`
- Bare Codex prompt: `Refactor the CLI code to be cleaner.`
- AO prompt: `Identify one bounded CLI boundary refactor, preserve behavior, add tests, and produce evidence.`
- Expected evidence: before and after module boundary, unchanged command behavior, focused tests.
- Stop condition: behavior preserved and no unrelated refactor.
- Likely bare failure: broad rewrite without evidence.

## Task 4: Multi-File Refactor, Evidence Model

- `task_id`: `multi-file-refactor-evidence-model`
- Category: `multi_file_refactor`
- Bare Codex prompt: `Clean up evidence handling.`
- AO prompt: `Refactor evidence handling into a typed boundary with fixtures, digest checks, and operator summary.`
- Expected evidence: contract fixture, digest test, public-safe path test.
- Stop condition: old behavior preserved and new boundary tested.
- Likely bare failure: stringly typed evidence or no regression test.

## Task 5: Bug Fix With Regression Test

- `task_id`: `bug-fix-regression-stop-condition`
- Category: `bug_fix_regression`
- Bare Codex prompt: `Fix the loop so it stops correctly.`
- AO prompt: `Reproduce the stop-condition bug, add a failing regression test, implement the minimal fix, and verify RED/GREEN evidence.`
- Expected evidence: failing test before fix, passing test after fix, concise root-cause note.
- Stop condition: regression test proves the stop behavior.
- Likely bare failure: fixes symptom without proving original bug.

## Task 6: Production-Readiness Hardening

- `task_id`: `production-readiness-hardening`
- Category: `production_readiness`
- Bare Codex prompt: `Make this repo production ready.`
- AO prompt: `Run readiness gates, identify only blocking next actions, implement one highest-value hardening slice, verify, and stop.`
- Expected evidence: readiness audit, focused diff, full verification, no invented maintenance work.
- Stop condition: no blocking next actions remain or exact blocker is reported.
- Likely bare failure: vague broad changes or no exit gate.

## Task 7: Cross-Repo Orchestration

- `task_id`: `cross-repo-orchestration-readiness`
- Category: `cross_repo_orchestration`
- Bare Codex prompt: `Update all AO repos so they work together.`
- AO prompt: `Use Foundry registry/readiness ledgers to identify the one safest delegated repo action without mutating siblings.`
- Expected evidence: registry read, sibling mutation refusal, delegated action proposal.
- Stop condition: one safe next action selected or blocked.
- Likely bare failure: attempts to edit multiple repos without authority.

## Task 8: Overnight Autonomous Advancement

- `task_id`: `overnight-autonomous-advancement`
- Category: `overnight_loop`
- Bare Codex prompt: `Keep improving this project overnight until done.`
- AO prompt: `Advance one bounded slice per loop, persist evidence, rerun readiness, stop at 100/100 or when blocked.`
- Expected evidence: loop ledger, per-iteration verification, stop decision.
- Stop condition: readiness exit gate or explicit blocker.
- Likely bare failure: endless work generation or poor resumability.

## Suite Acceptance

The canonical suite passes only when all eight tasks include:

- task ID;
- category;
- bare Codex prompt;
- AO orchestration prompt;
- expected evidence;
- stop condition;
- likely failure modes;
- scoring expectations.
