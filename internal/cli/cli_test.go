package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpListsAllCommands(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("Run(--help) exit code = %d, want 0", code)
	}

	help := stdout.String()
	for _, command := range []string{"suite", "competitor", "run", "evidence", "score", "compare", "report", "gate", "safety"} {
		if !strings.Contains(help, command) {
			t.Fatalf("help output missing command %q:\n%s", command, help)
		}
	}
}

func TestProductGateCommands(t *testing.T) {
	root := repoRoot(t)
	tmpDir := t.TempDir()

	cases := []struct {
		name string
		args []string
	}{
		{
			name: "suite validate",
			args: []string{"suite", "validate", "--suite", filepath.Join(root, "examples/suites/valid/ao-arena-v0.1.json")},
		},
		{
			name: "bare competitor validate",
			args: []string{"competitor", "validate", "--competitor", filepath.Join(root, "examples/competitors/valid/bare-codex.json")},
		},
		{
			name: "ao competitor validate",
			args: []string{"competitor", "validate", "--competitor", filepath.Join(root, "examples/competitors/valid/ao-orchestration.json")},
		},
		{
			name: "compare fixture",
			args: []string{"compare", "--suite", filepath.Join(root, "examples/suites/valid/ao-arena-v0.1.json"), "--fixture-mode", "--out", filepath.Join(tmpDir, "arena-report.json")},
		},
		{
			name: "report render",
			args: []string{"report", "render", "--report", filepath.Join(tmpDir, "arena-report.json"), "--out", filepath.Join(tmpDir, "arena-report.md")},
		},
		{
			name: "gate promotion",
			args: []string{"gate", "promotion", "--report", filepath.Join(tmpDir, "arena-report.json"), "--out", filepath.Join(tmpDir, "arena-promotion-gate.json")},
		},
		{
			name: "safety scan",
			args: []string{"safety", "scan", "--path", filepath.Join(root, "examples"), "--out", filepath.Join(tmpDir, "arena-safety-scan.json")},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run(%v) exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", tc.args, code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestInvalidFixturesFailClosed(t *testing.T) {
	root := repoRoot(t)
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "suite missing task",
			args: []string{"suite", "validate", "--suite", filepath.Join(root, "examples/suites/invalid/missing-task-id.json")},
			want: "unknown task",
		},
		{
			name: "live competitor without opt in",
			args: []string{"competitor", "validate", "--competitor", filepath.Join(root, "examples/competitors/invalid/live-without-opt-in.json")},
			want: "live mode requires operator opt-in",
		},
		{
			name: "unsafe evidence path",
			args: []string{"evidence", "validate", "--bundle", filepath.Join(root, "examples/evidence/invalid/local-absolute-path.json")},
			want: "local absolute path",
		},
		{
			name: "score above maximum",
			args: []string{"score", "--attempt", filepath.Join(root, "examples/attempts/valid/fixture-attempt.json"), "--scorecard", filepath.Join(root, "examples/scorecards/invalid/score-over-maximum.json"), "--out", filepath.Join(t.TempDir(), "score.json")},
			want: "score above maximum",
		},
		{
			name: "missing baseline",
			args: []string{"gate", "promotion", "--report", filepath.Join(root, "examples/reports/invalid/missing-baseline.json"), "--out", filepath.Join(t.TempDir(), "gate.json")},
			want: "baseline",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 {
				t.Fatalf("Run(%v) exit code = 0, want failure", tc.args)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr missing %q:\n%s", tc.want, stderr.String())
			}
		})
	}
}

func TestFixtureRunnerCreatesEightAttempts(t *testing.T) {
	root := repoRoot(t)
	out := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"run", "fixture",
		"--suite", filepath.Join(root, "examples/suites/valid/ao-arena-v0.1.json"),
		"--competitor", filepath.Join(root, "examples/competitors/valid/ao-orchestration.json"),
		"--out", out,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("fixture run failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	matches, err := filepath.Glob(filepath.Join(out, "attempts", "ao-arena-v0.1", "ao-orchestration", "*", "attempt.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 8 {
		t.Fatalf("attempt count = %d, want 8; matches=%v", len(matches), matches)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
