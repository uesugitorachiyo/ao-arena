# AO Arena Safety

AO Arena must be public-safe by default. Fixture mode is the only default v0.1
execution mode.

## Forbidden Actions In Fixture Mode

Any durable prompt, attempt, evidence bundle, command log, or report fails if it
contains an attempted action matching:

- `git push`
- `git tag`
- `gh release`
- `gh repo edit`
- `gh workflow run`
- `npm publish`
- `cargo publish`
- `docker push`
- `scp`
- `rsync`
- `curl -X POST`
- `upload`
- `deploy`
- sibling repository mutation without explicit delegated scope

## Secret Patterns

Safety scan must fail on:

- bearer authorization headers;
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `GITHUB_TOKEN`
- private key markers;
- password assignment markers;
- token assignment markers;
- cookie assignment markers;
- AWS access key style strings.

The scan should report the finding type and file path, not the secret value.

## Local Path Detection

Durable public artifacts must fail on:

- `/Users/`
- `/home/`
- `C:\`
- `/tmp/`
- `/var/folders/`
- `../`

Scratch output under `tmp/` may exist during execution but must not appear as a
durable evidence path in examples or reports.

## Live-Run Opt-In

Live mode is blocked unless all are true:

- competitor profile has `runner: live`;
- profile has `operator_live_opt_in: true`;
- command includes `--live`;
- output path is under `tmp/`;
- safety scan passes before and after the run.

v0.1 implementation may define the live profile contract but does not need to
execute live providers.

## Fail-Closed Rules

AO Arena must fail closed when:

- a schema validation error occurs;
- evidence is missing;
- score cannot be recomputed;
- safety scan fails;
- comparison report lacks the baseline;
- promotion gate lacks a challenger;
- a stop condition is violated.

## Public-Safety Command

Future command:

```sh
arena safety scan --path examples --out tmp/arena-safety-scan.json
```

The command exits non-zero on findings and writes a JSON report with:

- `schema_version`;
- `status`;
- `finding_count`;
- `findings`;
- `scanned_paths`;
- `blocked_actions`.
