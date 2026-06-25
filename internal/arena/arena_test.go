package arena

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScoreWorkedExamples(t *testing.T) {
	root := repoRoot(t)

	cases := []struct {
		path string
		want int
	}{
		{filepath.Join(root, "examples/scorecards/valid/bare-codex-worked.json"), 38},
		{filepath.Join(root, "examples/scorecards/valid/ao-orchestration-worked.json"), 95},
	}

	for _, tc := range cases {
		scorecard, err := LoadScorecard(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ComputeScore(scorecard)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("ComputeScore(%s) = %d, want %d", tc.path, got, tc.want)
		}
	}
}

func TestSafetyScanRedactsSecretsAndForbiddenActions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsafe.json")
	err := os.WriteFile(path, []byte(`{"command":"git push origin main","header":"Authorization: `+"Bearer fixture-value"+`"}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	report, err := ScanPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("safety status = %q, want failed", report.Status)
	}
	if report.FindingCount != 2 {
		t.Fatalf("finding count = %d, want 2", report.FindingCount)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret-value") {
		t.Fatalf("safety report leaked secret value: %s", raw)
	}
}

func TestInvalidTaskFixtureFailsValidation(t *testing.T) {
	root := repoRoot(t)
	err := ValidateTask(filepath.Join(root, "examples/tasks/invalid/missing-stop-condition.json"))
	if err == nil {
		t.Fatalf("ValidateTask accepted invalid task")
	}
	if !strings.Contains(err.Error(), "stop condition") {
		t.Fatalf("invalid task error = %v, want stop condition", err)
	}
}

func TestAttemptInvalidFixtureFailsSafetyScan(t *testing.T) {
	root := repoRoot(t)
	report, err := ScanPath(filepath.Join(root, "examples/attempts/invalid/unsafe-action.json"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("invalid attempt scan status = %q, want failed", report.Status)
	}
}

func TestFixtureRunnerRefusesDurablePublicOutput(t *testing.T) {
	root := repoRoot(t)
	_, err := RunFixture(
		filepath.Join(root, "examples/suites/valid/ao-arena-v0.1.json"),
		filepath.Join(root, "examples/competitors/valid/ao-orchestration.json"),
		filepath.Join(root, "examples", "generated-attempts"),
	)
	if err == nil {
		t.Fatalf("RunFixture accepted durable examples output")
	}
}

func TestPromotionGateRequiresFivePointSafeWin(t *testing.T) {
	pass := ComparisonReport{
		SchemaVersion: "ao.arena.comparison-report.v0.1",
		SuiteID:       "ao-arena-v0.1",
		Baseline:      CompetitorScore{CompetitorID: "bare-codex", AggregateScore: 80},
		Challengers:   []CompetitorScore{{CompetitorID: "ao-orchestration", AggregateScore: 85}},
		Winner:        "ao-orchestration",
		SafetyStatus:  "passed",
	}
	gate, err := EvaluatePromotion(pass)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Status != "passed" {
		t.Fatalf("gate status = %q, want passed", gate.Status)
	}

	tie := pass
	tie.Challengers[0].AggregateScore = 84
	gate, err = EvaluatePromotion(tie)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Status != "failed" {
		t.Fatalf("tie-ish gate status = %q, want failed", gate.Status)
	}
}

func TestPromotionGateRejectsUnsafeReport(t *testing.T) {
	report := ComparisonReport{
		SchemaVersion: "ao.arena.comparison-report.v0.1",
		SuiteID:       "ao-arena-v0.1",
		Baseline:      CompetitorScore{CompetitorID: "bare-codex", AggregateScore: 38},
		Challengers:   []CompetitorScore{{CompetitorID: "ao-orchestration", AggregateScore: 95}},
		Winner:        "ao-orchestration",
		SafetyStatus:  "failed",
	}
	gate, err := EvaluatePromotion(report)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Status != "failed" {
		t.Fatalf("unsafe gate status = %q, want failed", gate.Status)
	}
}

func TestImportHelpersTreatSourcesAsEvidenceOnlyAndFailClosed(t *testing.T) {
	root := repoRoot(t)
	for _, name := range []string{
		"active-stack-readiness.json",
		"ao2-run-summary.json",
		"covenant-policy-decision.json",
		"forge-packet-summary.json",
		"foundry-goalrun-readiness.json",
	} {
		result, err := ValidateImport(filepath.Join(root, "examples/imports/valid", name))
		if err != nil {
			t.Fatal(err)
		}
		if result.Authority != "evidence-input-only" {
			t.Fatalf("import authority = %q, want evidence-input-only", result.Authority)
		}
	}

	_, err := ValidateImport(filepath.Join(root, "examples/imports/valid/missing-source.json"))
	if err == nil {
		t.Fatalf("ValidateImport accepted missing source")
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
