package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ao-foundry/ao-arena/internal/arena"
)

func TestCompareRealAttemptsCommand(t *testing.T) {
	input := writeCLIRealAttemptFixture(t)
	out := filepath.Join(t.TempDir(), "comparison.json")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"compare", "real-attempts", "--input", input, "--out", out}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("real-attempt comparison failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "winner=ao-orchestration") || strings.Contains(stdout.String(), input) || strings.Contains(stdout.String(), out) {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatal(err)
	}
	if report["schema_version"] != "ao.arena.real-attempt-comparison.v0.1" || report["result"] != "strong_win" {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestCompareRealAttemptsCommandRejectsLooseFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing input", args: []string{"compare", "real-attempts", "--out", "comparison.json"}, want: "usage:"},
		{name: "unknown flag", args: []string{"compare", "real-attempts", "--input", "manifest.json", "--out", "comparison.json", "--extra", "value"}, want: "unknown flag"},
		{name: "duplicate input", args: []string{"compare", "real-attempts", "--input", "one.json", "--input", "two.json", "--out", "comparison.json"}, want: "duplicate --input"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tc.args, &stdout, &stderr)
			if code == 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("Run(%v) code=%d stderr=%q, want %q", tc.args, code, stderr.String(), tc.want)
			}
		})
	}
}

func TestHelpDocumentsRealAttemptComparator(t *testing.T) {
	var stdout bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "compare real-attempts --input <manifest.json> --out <comparison.json>") {
		t.Fatalf("help missing real-attempt command:\n%s", stdout.String())
	}
}

func writeCLIRealAttemptFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")
	if err := os.Mkdir(evidenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	pairs := make([]any, 0, 10)
	tasks := make([]arena.RealAttemptTaskContract, 0, 10)
	for i := 0; i < 10; i++ {
		taskID := fmt.Sprintf("cli-real-task-%02d", i)
		snapshot := fmt.Sprintf("%064x", i+1)
		bare := cliRealAttempt(t, evidenceDir, "bare-codex", fmt.Sprintf("bare-cli-%02d", i), taskID, snapshot, 50)
		ao := cliRealAttempt(t, evidenceDir, "ao-orchestration", fmt.Sprintf("ao-cli-%02d", i), taskID, snapshot, 65)
		pairs = append(pairs, map[string]any{"bare_codex": bare, "ao_orchestration": ao})
		tasks = append(tasks, arena.RealAttemptTaskContract{
			TaskID:                  taskID,
			SnapshotSHA256:          snapshot,
			ExpectedTerminal:        "verified_closure",
			VerifierCommandSHA256:   cliDigest("bounded-verifier:" + taskID),
			AuthorityBoundarySHA256: cliDigest("authority-boundary:" + taskID),
		})
	}
	portfolioBody, err := json.MarshalIndent(arena.RealAttemptTaskPortfolio{
		SchemaVersion: "ao.arena.real-attempt-task-portfolio.v0.1",
		Tasks:         tasks,
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	portfolioBody = append(portfolioBody, '\n')
	portfolioDigest := sha256.Sum256(portfolioBody)
	if err := os.WriteFile(filepath.Join(dir, "task-portfolio.json"), portfolioBody, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schema_version": "ao.arena.real-attempt-manifest.v0.1",
		"suite_id":       "cli-real-attempt-suite",
		"task_portfolio": map[string]any{"path": "task-portfolio.json", "sha256": hex.EncodeToString(portfolioDigest[:])},
		"pairs":          pairs,
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(input, append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return input
}

func cliRealAttempt(t *testing.T, evidenceDir, competitorID, attemptID, taskID, snapshot string, score int) map[string]any {
	t.Helper()
	scorecard := arena.Scorecard{
		SchemaVersion:  "ao.arena.scorecard.v0.1",
		CompetitorID:   competitorID,
		TaskID:         taskID,
		CategoryScores: cliCategoryScores(score),
		Penalties:      []arena.Penalty{},
		Score:          score,
		SafetyStatus:   "passed",
		Derivation:     "bounded CLI test score",
	}
	sourceResult := arena.RealAttemptSourceResult{
		Status:              "completed",
		StopConditionStatus: "satisfied",
		SafetyStatus:        "passed",
		Scorecard:           scorecard,
		Regressions:         []string{},
		Limitations:         []string{"provider execution excluded"},
	}
	sourceBody, err := json.Marshal(sourceResult)
	if err != nil {
		t.Fatal(err)
	}
	sourceDigest := sha256.Sum256(sourceBody)
	evidence := arena.RealAttemptEvidence{
		SchemaVersion:    "ao.arena.real-attempt-evidence.v0.1",
		AttemptID:        attemptID,
		TaskID:           taskID,
		CompetitorID:     competitorID,
		SnapshotSHA256:   snapshot,
		ExpectedTerminal: "verified_closure",
		Verification: arena.RealAttemptVerification{
			Status:                  "passed",
			VerifierCommandSHA256:   cliDigest("bounded-verifier:" + taskID),
			AuthorityBoundarySHA256: cliDigest("authority-boundary:" + taskID),
			VerifierExitCode:        cliIntPointer(0),
			SourceResultSHA256:      hex.EncodeToString(sourceDigest[:]),
			AuthorityChecked:        cliBoolPointer(true),
			EvidenceRetained:        cliBoolPointer(true),
			PublicWrites:            cliIntPointer(0),
			UnsupportedClaims:       cliIntPointer(0),
		},
		SourceResult: sourceResult,
	}
	evidenceBody, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	evidenceBody = append(evidenceBody, '\n')
	digest := sha256.Sum256(evidenceBody)
	evidenceName := attemptID + ".json"
	if err := os.WriteFile(filepath.Join(evidenceDir, evidenceName), evidenceBody, 0o600); err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"attempt_id":            attemptID,
		"task_id":               taskID,
		"competitor_id":         competitorID,
		"snapshot_sha256":       snapshot,
		"expected_terminal":     "verified_closure",
		"status":                "completed",
		"stop_condition_status": "satisfied",
		"evidence": map[string]any{
			"path":   filepath.ToSlash(filepath.Join("evidence", evidenceName)),
			"sha256": hex.EncodeToString(digest[:]),
		},
		"scorecard":   scorecard,
		"regressions": []any{},
		"limitations": []any{"provider execution excluded"},
	}
}

func cliDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func cliIntPointer(value int) *int { return &value }

func cliBoolPointer(value bool) *bool { return &value }

func cliCategoryScores(total int) map[string]int {
	maxima := []struct {
		name string
		max  int
	}{
		{"correctness", 20},
		{"test_quality", 15},
		{"evidence_quality", 15},
		{"decomposition_quality", 10},
		{"safety_policy", 15},
		{"resumability", 10},
		{"stop_condition_accuracy", 10},
		{"operator_handoff", 5},
	}
	result := make(map[string]int, len(maxima))
	for _, category := range maxima {
		value := min(total, category.max)
		result[category.name] = value
		total -= value
	}
	return result
}
