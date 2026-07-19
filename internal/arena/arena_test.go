package arena

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

func TestSafetyScanRejectsSymlinkedSafeSuffixFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-dependent on Windows")
	}
	dir := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outsidePath, []byte(`{"safe":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(dir, "linked.json")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	_, err := ScanPath(dir)

	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("ScanPath symlink error = %v, want symlink diagnostic", err)
	}
}

func TestSafetyScanRejectsFileCountLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= maxSafetyScanFiles; i++ {
		path := filepath.Join(dir, fmt.Sprintf("safe-%05d.txt", i))
		if err := os.WriteFile(path, []byte("safe\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	_, err := ScanPath(dir)

	if err == nil || !strings.Contains(err.Error(), "file count limit") {
		t.Fatalf("ScanPath file-count error = %v, want file-count diagnostic", err)
	}
}

func TestSafetyScanRejectsOversizedSafeSuffixFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oversized.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("A", maxSafetyScanFileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ScanPath(dir)

	if err == nil || !strings.Contains(err.Error(), "file size limit") {
		t.Fatalf("ScanPath oversized-file error = %v, want file-size diagnostic", err)
	}
}

func TestSafetyScanRejectsTotalByteLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i <= maxSafetyScanTotalBytes/maxSafetyScanFileBytes; i++ {
		path := filepath.Join(dir, fmt.Sprintf("safe-total-%05d.txt", i))
		if err := os.WriteFile(path, []byte(strings.Repeat("A", maxSafetyScanFileBytes)), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	_, err := ScanPath(dir)

	if err == nil || !strings.Contains(err.Error(), "total byte limit") {
		t.Fatalf("ScanPath total-byte error = %v, want total-byte diagnostic", err)
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

func TestGitHubIssueMonth2ReproductionEvaluationMeetsTruthSetThresholds(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "examples", "reports", "valid", "github-issue-month2-reproduction-evaluation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatal(err)
	}
	if report["schema_version"] != "ao.arena.github-issue-reproduction-evaluation.v0.1" ||
		report["status"] != "passed" {
		t.Fatalf("unexpected reproduction evaluation identity: %#v", report)
	}
	thresholds := report["thresholds"].(map[string]any)
	metrics := report["metrics"].(map[string]any)
	if metrics["precision"].(float64) < thresholds["precision_minimum"].(float64) ||
		metrics["recall"].(float64) < thresholds["recall_minimum"].(float64) {
		t.Fatalf("reproduction metrics missed thresholds: thresholds=%#v metrics=%#v", thresholds, metrics)
	}
	if metrics["truth_set_size"].(float64) != 13 ||
		metrics["authentic_bug_fixtures"].(float64) != 2 ||
		metrics["non_bug_fixtures"].(float64) != 11 {
		t.Fatalf("truth-set metrics drifted: %#v", metrics)
	}
	flaky := report["flaky_reproduction"].(map[string]any)
	if flaky["runs"].(float64) < 10 ||
		flaky["confidence"].(float64) < thresholds["flaky_confidence_minimum"].(float64) ||
		flaky["uncertainty_promoted_to_authentic_without_measurement"] != false {
		t.Fatalf("flaky reproduction confidence is not measured: %#v", flaky)
	}
	for _, key := range []string{
		"negative_controls_passed",
	} {
		if report[key] != true {
			t.Fatalf("%s = %#v, want true", key, report[key])
		}
	}
	for _, key := range []string{
		"security_sensitive_public_repair_entered",
		"provider_call_performed",
		"promotion_requested",
		"release_selected",
	} {
		if report[key] != false {
			t.Fatalf("%s = %#v, want false", key, report[key])
		}
	}
}

func TestGitHubIssueMonth3RepairCandidateEvaluationRejectsFalseFixes(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "examples", "reports", "valid", "github-issue-month3-repair-candidate-evaluation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(body, &report); err != nil {
		t.Fatal(err)
	}
	if report["schema_version"] != "ao.arena.github-issue-repair-candidate-evaluation.v0.1" ||
		report["status"] != "passed" ||
		report["accepted_candidate"] != "candidate_minimal_fixture_patch" {
		t.Fatalf("unexpected repair candidate evaluation identity: %#v", report)
	}
	metrics := report["accepted_candidate_metrics"].(map[string]any)
	for _, key := range []string{"pre_patch_failure_preserved", "post_patch_regression_passed", "negative_control_passed", "rollback_verified", "replay_digest_match"} {
		if metrics[key] != true {
			t.Fatalf("accepted candidate metric %s = %#v, want true", key, metrics[key])
		}
	}
	if metrics["unrelated_files_changed"].(float64) != 0 {
		t.Fatalf("accepted candidate changed unrelated files: %#v", metrics)
	}
	rejected := report["rejected_candidates"].([]any)
	if len(rejected) != 2 {
		t.Fatalf("expected two rejected false-fix candidates: %#v", rejected)
	}
	for _, key := range []string{"test_deleted_or_weakened_rejected", "lint_or_policy_disabled_rejected", "unrelated_file_change_rejected", "security_sensitive_public_repair_rejected", "provider_execution_required_rejected"} {
		if report["false_fix_rejection"].(map[string]any)[key] != true {
			t.Fatalf("missing false-fix rejection %s: %#v", key, report["false_fix_rejection"])
		}
	}
	for _, key := range []string{"feature_generated_draft_pr_opened", "issue_write_performed", "release_selected", "rsi_authorized"} {
		if report["denied_actions"].(map[string]any)[key] != false {
			t.Fatalf("denied action %s must remain false: %#v", key, report["denied_actions"])
		}
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
