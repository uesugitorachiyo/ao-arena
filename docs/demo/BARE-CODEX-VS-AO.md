# Bare Codex vs AO Orchestration

This public demo pack runs entirely in AO Arena fixture mode. It does not need
live credentials and does not call model providers.

Run:

```sh
arena suite validate --suite examples/suites/valid/ao-arena-v0.1.json
arena compare --suite examples/suites/valid/ao-arena-v0.1.json --fixture-mode --out tmp/arena-report.json
arena report render --report tmp/arena-report.json --out tmp/arena-report.md
arena gate promotion --report tmp/arena-report.json --out tmp/arena-promotion-gate.json
```

AO orchestration wins the canonical fixture comparison when its evidence proves
better correctness, test quality, public safety, resumability, and stop
condition accuracy than the bare prompt baseline.
