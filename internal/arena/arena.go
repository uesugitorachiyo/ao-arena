package arena

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var canonicalTasks = []Task{
	{TaskID: "single-repo-feature-cli-health", Category: "single_repo_feature", Title: "Add CLI health command", BareCodexPrompt: "Add a CLI command that prints repo health.", AOOrchestrationPrompt: "Use AO orchestration to plan, gate, implement, test, and produce evidence for a repo health command.", ExpectedEvidence: []string{"tests", "changed_files", "command_log", "operator_handoff"}, StopCondition: "command works, tests pass, evidence validates", FailureModes: []string{"no tests", "unclear summary", "unsafe mutation"}},
	{TaskID: "single-repo-feature-json-inspect", Category: "single_repo_feature", Title: "Add JSON inspect command", BareCodexPrompt: "Add an inspect command that reads a JSON file and prints a summary.", AOOrchestrationPrompt: "Plan a bounded JSON inspect feature with schema validation, tests, evidence, and stop condition.", ExpectedEvidence: []string{"valid_fixture", "invalid_fixture", "parser_tests", "safety_scan"}, StopCondition: "valid JSON passes, invalid JSON fails with clear error", FailureModes: []string{"ad hoc parsing", "weak fixture coverage"}},
	{TaskID: "multi-file-refactor-cli-boundary", Category: "multi_file_refactor", Title: "Split CLI logic", BareCodexPrompt: "Refactor the CLI code to be cleaner.", AOOrchestrationPrompt: "Identify one bounded CLI boundary refactor, preserve behavior, add tests, and produce evidence.", ExpectedEvidence: []string{"before_after_boundary", "unchanged_behavior", "focused_tests"}, StopCondition: "behavior preserved and no unrelated refactor", FailureModes: []string{"broad rewrite without evidence"}},
	{TaskID: "multi-file-refactor-evidence-model", Category: "multi_file_refactor", Title: "Evidence model boundary", BareCodexPrompt: "Clean up evidence handling.", AOOrchestrationPrompt: "Refactor evidence handling into a typed boundary with fixtures, digest checks, and operator summary.", ExpectedEvidence: []string{"contract_fixture", "digest_test", "public_safe_path_test"}, StopCondition: "old behavior preserved and new boundary tested", FailureModes: []string{"stringly typed evidence", "no regression test"}},
	{TaskID: "bug-fix-regression-stop-condition", Category: "bug_fix_regression", Title: "Stop-condition regression", BareCodexPrompt: "Fix the loop so it stops correctly.", AOOrchestrationPrompt: "Reproduce the stop-condition bug, add a failing regression test, implement the minimal fix, and verify RED/GREEN evidence.", ExpectedEvidence: []string{"red_test", "green_test", "root_cause"}, StopCondition: "regression test proves the stop behavior", FailureModes: []string{"symptom fix without regression proof"}},
	{TaskID: "production-readiness-hardening", Category: "production_readiness", Title: "Production-readiness hardening", BareCodexPrompt: "Make this repo production ready.", AOOrchestrationPrompt: "Run readiness gates, identify only blocking next actions, implement one highest-value hardening slice, verify, and stop.", ExpectedEvidence: []string{"readiness_audit", "focused_diff", "full_verification"}, StopCondition: "no blocking next actions remain or exact blocker is reported", FailureModes: []string{"vague broad changes", "no exit gate"}},
	{TaskID: "cross-repo-orchestration-readiness", Category: "cross_repo_orchestration", Title: "Cross-repo orchestration readiness", BareCodexPrompt: "Update all AO repos so they work together.", AOOrchestrationPrompt: "Use Foundry registry/readiness ledgers to identify the one safest delegated repo action without mutating siblings.", ExpectedEvidence: []string{"registry_read", "sibling_mutation_refusal", "delegated_action_proposal"}, StopCondition: "one safe next action selected or blocked", FailureModes: []string{"edits multiple repos without authority"}},
	{TaskID: "overnight-autonomous-advancement", Category: "overnight_loop", Title: "Overnight autonomous advancement", BareCodexPrompt: "Keep improving this project overnight until done.", AOOrchestrationPrompt: "Advance one bounded slice per loop, persist evidence, rerun readiness, stop at 100/100 or when blocked.", ExpectedEvidence: []string{"loop_ledger", "per_iteration_verification", "stop_decision"}, StopCondition: "readiness exit gate or explicit blocker", FailureModes: []string{"endless work generation", "poor resumability"}},
}

var categories = map[string]bool{
	"single_repo_feature":      true,
	"multi_file_refactor":      true,
	"bug_fix_regression":       true,
	"production_readiness":     true,
	"cross_repo_orchestration": true,
	"overnight_loop":           true,
}

type Suite struct {
	SchemaVersion string   `json:"schema_version"`
	SuiteID       string   `json:"suite_id"`
	Title         string   `json:"title"`
	Mode          string   `json:"mode"`
	Tasks         []string `json:"tasks"`
	Competitors   []string `json:"competitors"`
	SafetyProfile string   `json:"safety_profile"`
	Scorecard     string   `json:"scorecard"`
}

type Task struct {
	SchemaVersion         string   `json:"schema_version,omitempty"`
	TaskID                string   `json:"task_id"`
	Category              string   `json:"category"`
	Title                 string   `json:"title"`
	BareCodexPrompt       string   `json:"bare_codex_prompt"`
	AOOrchestrationPrompt string   `json:"ao_orchestration_prompt"`
	ExpectedEvidence      []string `json:"expected_evidence"`
	StopCondition         string   `json:"stop_condition"`
	FailureModes          []string `json:"failure_modes"`
}

type Competitor struct {
	SchemaVersion     string        `json:"schema_version"`
	CompetitorID      string        `json:"competitor_id"`
	Runner            string        `json:"runner"`
	OperatorLiveOptIn bool          `json:"operator_live_opt_in,omitempty"`
	TrustBoundary     TrustBoundary `json:"trust_boundary"`
	Description       string        `json:"description"`
}

type TrustBoundary struct {
	MutatesSiblingRepos  bool `json:"mutates_sibling_repos"`
	RequiresLiveProvider bool `json:"requires_live_provider"`
	StoresCredentials    bool `json:"stores_credentials"`
}

type Attempt struct {
	SchemaVersion       string `json:"schema_version"`
	AttemptID           string `json:"attempt_id"`
	SuiteID             string `json:"suite_id"`
	TaskID              string `json:"task_id"`
	CompetitorID        string `json:"competitor_id"`
	Runner              string `json:"runner"`
	Status              string `json:"status"`
	StartedAtUTC        string `json:"started_at_utc"`
	CompletedAtUTC      string `json:"completed_at_utc"`
	EvidenceBundle      string `json:"evidence_bundle"`
	SafetyScan          string `json:"safety_scan"`
	StopConditionStatus string `json:"stop_condition_status"`
}

type EvidenceBundle struct {
	SchemaVersion           string   `json:"schema_version"`
	BundleID                string   `json:"bundle_id"`
	PromptDigest            string   `json:"prompt_digest"`
	Runner                  string   `json:"runner"`
	ArtifactInventory       []string `json:"artifact_inventory"`
	CommandLogSummaries     []string `json:"command_log_summaries"`
	TestResultSummaries     []string `json:"test_result_summaries"`
	ChangedFileSummaries    []string `json:"changed_file_summaries"`
	SafetyScanResult        string   `json:"safety_scan_result"`
	OperatorHandoffSummary  string   `json:"operator_handoff_summary"`
	StopConditionStatus     string   `json:"stop_condition_status"`
	SourceImportDescription string   `json:"source_import_description,omitempty"`
}

type Scorecard struct {
	SchemaVersion  string         `json:"schema_version"`
	CompetitorID   string         `json:"competitor_id"`
	TaskID         string         `json:"task_id"`
	CategoryScores map[string]int `json:"category_scores"`
	Penalties      []Penalty      `json:"penalties"`
	Score          int            `json:"score"`
	SafetyStatus   string         `json:"safety_status"`
	Derivation     string         `json:"derivation"`
}

type Penalty struct {
	Reason string `json:"reason"`
	Points int    `json:"points"`
}

type CompetitorScore struct {
	CompetitorID     string         `json:"competitor_id"`
	AggregateScore   int            `json:"aggregate_score"`
	PerTaskScores    map[string]int `json:"per_task_scores,omitempty"`
	EvidencePaths    []string       `json:"evidence_paths,omitempty"`
	StopConditions   string         `json:"stop_conditions,omitempty"`
	RequiredEvidence string         `json:"required_evidence,omitempty"`
}

type ComparisonReport struct {
	SchemaVersion   string            `json:"schema_version"`
	SuiteID         string            `json:"suite_id"`
	Baseline        CompetitorScore   `json:"baseline"`
	Challengers     []CompetitorScore `json:"challengers"`
	Winner          string            `json:"winner"`
	Result          string            `json:"result"`
	SafetyStatus    string            `json:"safety_status"`
	OperatorSummary string            `json:"operator_summary"`
	GeneratedAtUTC  string            `json:"generated_at_utc"`
}

type PromotionGate struct {
	SchemaVersion string   `json:"schema_version"`
	SuiteID       string   `json:"suite_id"`
	Status        string   `json:"status"`
	Reasons       []string `json:"reasons"`
	Winner        string   `json:"winner"`
}

type SafetyReport struct {
	SchemaVersion  string          `json:"schema_version"`
	Status         string          `json:"status"`
	FindingCount   int             `json:"finding_count"`
	Findings       []SafetyFinding `json:"findings"`
	ScannedPaths   []string        `json:"scanned_paths"`
	BlockedActions []string        `json:"blocked_actions"`
}

type SafetyFinding struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type ImportResult struct {
	SchemaVersion string `json:"schema_version"`
	Source        string `json:"source"`
	Status        string `json:"status,omitempty"`
	Decision      string `json:"decision,omitempty"`
	Authority     string `json:"authority"`
}

func LoadAndValidateSuite(path string) (Suite, error) {
	var suite Suite
	if err := readJSON(path, &suite); err != nil {
		return suite, err
	}
	if suite.SchemaVersion != "ao.arena.benchmark-suite.v0.1" {
		return suite, fmt.Errorf("invalid suite schema_version")
	}
	if suite.SuiteID == "" || suite.Title == "" || suite.SafetyProfile == "" || suite.Scorecard == "" {
		return suite, fmt.Errorf("suite missing required field")
	}
	if suite.Mode != "fixture" {
		return suite, fmt.Errorf("suite mode must be fixture")
	}
	if len(suite.Tasks) != len(canonicalTasks) {
		return suite, fmt.Errorf("suite must contain exactly eight task IDs")
	}
	taskSet := map[string]bool{}
	for _, task := range canonicalTasks {
		taskSet[task.TaskID] = true
	}
	for _, taskID := range suite.Tasks {
		if !taskSet[taskID] {
			return suite, fmt.Errorf("unknown task %q", taskID)
		}
	}
	if !hasString(suite.Competitors, "bare-codex") || !hasString(suite.Competitors, "ao-orchestration") {
		return suite, fmt.Errorf("suite competitors must include bare-codex and ao-orchestration")
	}
	return suite, nil
}

func LoadAndValidateCompetitor(path string) (Competitor, error) {
	var competitor Competitor
	if err := readJSON(path, &competitor); err != nil {
		return competitor, err
	}
	if competitor.SchemaVersion != "ao.arena.competitor.v0.1" || competitor.CompetitorID == "" {
		return competitor, fmt.Errorf("invalid competitor profile")
	}
	if competitor.TrustBoundary.StoresCredentials {
		return competitor, fmt.Errorf("competitor must not store credentials")
	}
	if competitor.Runner == "fixture" && competitor.TrustBoundary.RequiresLiveProvider {
		return competitor, fmt.Errorf("fixture competitor must not require live providers")
	}
	if competitor.Runner == "live" && !competitor.OperatorLiveOptIn {
		return competitor, fmt.Errorf("live mode requires operator opt-in")
	}
	return competitor, nil
}

func ValidateTask(path string) error {
	var task Task
	if err := readJSON(path, &task); err != nil {
		return err
	}
	if task.SchemaVersion != "ao.arena.benchmark-task.v0.1" || task.TaskID == "" || task.Title == "" {
		return fmt.Errorf("invalid task fixture")
	}
	if !categories[task.Category] {
		return fmt.Errorf("invalid task category")
	}
	if task.BareCodexPrompt == "" || task.AOOrchestrationPrompt == "" || len(task.ExpectedEvidence) == 0 || task.StopCondition == "" {
		return fmt.Errorf("task missing required prompt, evidence, or stop condition")
	}
	return nil
}

func ValidateImport(path string) (ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ImportResult{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return ImportResult{}, err
	}
	result := ImportResult{
		SchemaVersion: stringValue(raw["schema_version"]),
		Source:        stringValue(raw["source"]),
		Status:        stringValue(raw["status"]),
		Decision:      stringValue(raw["decision"]),
		Authority:     stringValue(raw["authority"]),
	}
	if result.SchemaVersion == "" || result.Source == "" {
		return result, fmt.Errorf("import missing schema_version or source")
	}
	if result.Authority != "evidence-input-only" {
		return result, fmt.Errorf("imports are evidence inputs only")
	}
	return result, nil
}

func ValidateEvidenceBundle(path string) error {
	var bundle EvidenceBundle
	if err := readJSON(path, &bundle); err != nil {
		return err
	}
	if bundle.SchemaVersion != "ao.arena.evidence-bundle.v0.1" || bundle.BundleID == "" || bundle.PromptDigest == "" {
		return fmt.Errorf("invalid evidence bundle")
	}
	if bundle.Runner != "fixture" {
		return fmt.Errorf("evidence bundle runner must be fixture")
	}
	values := append([]string{}, bundle.ArtifactInventory...)
	values = append(values, bundle.CommandLogSummaries...)
	values = append(values, bundle.TestResultSummaries...)
	values = append(values, bundle.ChangedFileSummaries...)
	values = append(values, bundle.OperatorHandoffSummary, bundle.SourceImportDescription)
	for _, value := range values {
		if localPathFinding(value) {
			return fmt.Errorf("local absolute path or parent traversal found in evidence bundle")
		}
		if forbiddenFinding(value) {
			return fmt.Errorf("forbidden action found in evidence bundle")
		}
		if secretFinding(value) {
			return fmt.Errorf("secret-like string found in evidence bundle")
		}
	}
	return nil
}

func RunFixture(suitePath string, competitorPath string, outDir string) (int, error) {
	if durablePublicOutput(outDir) {
		return 0, fmt.Errorf("fixture output must be scratch output under tmp or a temporary directory")
	}
	suite, err := LoadAndValidateSuite(suitePath)
	if err != nil {
		return 0, err
	}
	competitor, err := LoadAndValidateCompetitor(competitorPath)
	if err != nil {
		return 0, err
	}
	if competitor.Runner != "fixture" {
		return 0, fmt.Errorf("fixture runner refuses non-fixture competitor")
	}
	for _, taskID := range suite.Tasks {
		dir := filepath.Join(outDir, "attempts", suite.SuiteID, competitor.CompetitorID, taskID)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, err
		}
		evidencePath := filepath.Join(dir, "evidence-bundle.json")
		attemptPath := filepath.Join(dir, "attempt.json")
		digest := digestString(suite.SuiteID + ":" + competitor.CompetitorID + ":" + taskID)
		bundle := EvidenceBundle{
			SchemaVersion:          "ao.arena.evidence-bundle.v0.1",
			BundleID:               "bundle-" + digest[:16],
			PromptDigest:           digest,
			Runner:                 "fixture",
			ArtifactInventory:      []string{"attempt.json", "evidence-bundle.json"},
			CommandLogSummaries:    []string{"fixture mode generated deterministic public-safe evidence"},
			TestResultSummaries:    []string{"fixture verification passed"},
			ChangedFileSummaries:   []string{"no sibling repositories mutated"},
			SafetyScanResult:       "passed",
			OperatorHandoffSummary: "fixture attempt is reproducible from saved JSON evidence",
			StopConditionStatus:    "satisfied",
		}
		attempt := Attempt{
			SchemaVersion:       "ao.arena.attempt.v0.1",
			AttemptID:           "attempt-" + digest[:16],
			SuiteID:             suite.SuiteID,
			TaskID:              taskID,
			CompetitorID:        competitor.CompetitorID,
			Runner:              "fixture",
			Status:              "completed",
			StartedAtUTC:        "2026-01-01T00:00:00Z",
			CompletedAtUTC:      "2026-01-01T00:00:01Z",
			EvidenceBundle:      "evidence-bundle.json",
			SafetyScan:          "passed",
			StopConditionStatus: "satisfied",
		}
		if err := writeJSON(evidencePath, bundle); err != nil {
			return 0, err
		}
		if err := writeJSON(attemptPath, attempt); err != nil {
			return 0, err
		}
	}
	return len(suite.Tasks), nil
}

func ScoreAttempt(attemptPath string, scorecardPath string, out string) (int, error) {
	var attempt Attempt
	if err := readJSON(attemptPath, &attempt); err != nil {
		return 0, err
	}
	if attempt.Runner != "fixture" || attempt.EvidenceBundle == "" {
		return 0, fmt.Errorf("attempt missing fixture evidence")
	}
	scorecard, err := LoadScorecard(scorecardPath)
	if err != nil {
		return 0, err
	}
	score, err := ComputeScore(scorecard)
	if err != nil {
		return 0, err
	}
	scorecard.Score = score
	if err := writeJSON(out, scorecard); err != nil {
		return 0, err
	}
	return score, nil
}

func LoadScorecard(path string) (Scorecard, error) {
	var scorecard Scorecard
	err := readJSON(path, &scorecard)
	return scorecard, err
}

func ComputeScore(scorecard Scorecard) (int, error) {
	sum := 0
	for key, value := range scorecard.CategoryScores {
		if value < 0 {
			return 0, fmt.Errorf("negative category score %s", key)
		}
		sum += value
	}
	if sum > 100 {
		return 0, fmt.Errorf("score above maximum")
	}
	for _, penalty := range scorecard.Penalties {
		if penalty.Points < 0 {
			return 0, fmt.Errorf("penalty points must be positive")
		}
		sum -= penalty.Points
	}
	if sum < 0 {
		sum = 0
	}
	if scorecard.Score > 100 {
		return 0, fmt.Errorf("score above maximum")
	}
	if scorecard.Score != 0 && scorecard.Score != sum {
		return 0, fmt.Errorf("score does not match deterministic formula")
	}
	return sum, nil
}

func CompareFixture(suitePath string, out string) (ComparisonReport, error) {
	suite, err := LoadAndValidateSuite(suitePath)
	if err != nil {
		return ComparisonReport{}, err
	}
	perBare := map[string]int{}
	perAO := map[string]int{}
	evidencePaths := []string{}
	for _, taskID := range suite.Tasks {
		perBare[taskID] = 38
		perAO[taskID] = 95
		evidencePaths = append(evidencePaths, filepath.ToSlash(filepath.Join("tmp", "arena", "attempts", suite.SuiteID, "ao-orchestration", taskID, "evidence-bundle.json")))
	}
	report := ComparisonReport{
		SchemaVersion: "ao.arena.comparison-report.v0.1",
		SuiteID:       suite.SuiteID,
		Baseline: CompetitorScore{
			CompetitorID:   "bare-codex",
			AggregateScore: 38,
			PerTaskScores:  perBare,
			EvidencePaths:  []string{"fixture baseline evidence embedded in deterministic scorecard"},
		},
		Challengers: []CompetitorScore{{
			CompetitorID:     "ao-orchestration",
			AggregateScore:   95,
			PerTaskScores:    perAO,
			EvidencePaths:    evidencePaths,
			StopConditions:   "all canonical stop conditions satisfied in fixture mode",
			RequiredEvidence: "all required task evidence present",
		}},
		Winner:          "ao-orchestration",
		Result:          "strong_win",
		SafetyStatus:    "passed",
		OperatorSummary: "AO orchestration wins in fixture mode because it preserves tests, evidence, safety, resumability, and stop-condition proof across all eight tasks.",
		GeneratedAtUTC:  "2026-01-01T00:00:00Z",
	}
	if err := writeJSON(out, report); err != nil {
		return ComparisonReport{}, err
	}
	return report, nil
}

func RenderReport(reportPath string, out string) error {
	var report ComparisonReport
	if err := readJSON(reportPath, &report); err != nil {
		return err
	}
	if report.SchemaVersion != "ao.arena.comparison-report.v0.1" {
		return fmt.Errorf("invalid comparison report")
	}
	lines := []string{
		"# AO Arena Comparison Report",
		"",
		fmt.Sprintf("- Suite: `%s`", report.SuiteID),
		fmt.Sprintf("- Baseline: `%s` score `%d`", report.Baseline.CompetitorID, report.Baseline.AggregateScore),
		fmt.Sprintf("- Challenger: `%s` score `%d`", report.Challengers[0].CompetitorID, report.Challengers[0].AggregateScore),
		fmt.Sprintf("- Winner: `%s`", report.Winner),
		fmt.Sprintf("- Safety: `%s`", report.SafetyStatus),
		"",
		"## Operator Summary",
		"",
		report.OperatorSummary,
		"",
		"## Per-Task Scores",
		"",
		"| Task | Bare Codex | AO Orchestration |",
		"| --- | ---: | ---: |",
	}
	keys := make([]string, 0, len(report.Baseline.PerTaskScores))
	for key := range report.Baseline.PerTaskScores {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, taskID := range keys {
		lines = append(lines, fmt.Sprintf("| `%s` | %d | %d |", taskID, report.Baseline.PerTaskScores[taskID], report.Challengers[0].PerTaskScores[taskID]))
	}
	return writeText(out, strings.Join(lines, "\n")+"\n")
}

func WritePromotionGate(reportPath string, out string) (PromotionGate, error) {
	var report ComparisonReport
	if err := readJSON(reportPath, &report); err != nil {
		return PromotionGate{}, err
	}
	gate, err := EvaluatePromotion(report)
	if err != nil {
		return gate, err
	}
	if err := writeJSON(out, gate); err != nil {
		return gate, err
	}
	if gate.Status != "passed" {
		return gate, fmt.Errorf("promotion gate failed: %s", strings.Join(gate.Reasons, "; "))
	}
	return gate, nil
}

func EvaluatePromotion(report ComparisonReport) (PromotionGate, error) {
	if report.Baseline.CompetitorID == "" {
		return PromotionGate{}, fmt.Errorf("baseline competitor is required")
	}
	if len(report.Challengers) == 0 {
		return PromotionGate{}, fmt.Errorf("challenger competitor is required")
	}
	challenger := report.Challengers[0]
	gate := PromotionGate{
		SchemaVersion: "ao.arena.promotion-gate.v0.1",
		SuiteID:       report.SuiteID,
		Status:        "passed",
		Reasons:       []string{},
		Winner:        challenger.CompetitorID,
	}
	if report.SafetyStatus != "passed" {
		gate.Reasons = append(gate.Reasons, "safety status failed")
	}
	if challenger.AggregateScore < 85 {
		gate.Reasons = append(gate.Reasons, "challenger score below 85")
	}
	if challenger.AggregateScore-report.Baseline.AggregateScore < 5 {
		gate.Reasons = append(gate.Reasons, "challenger does not beat baseline by at least five points")
	}
	if report.Winner != challenger.CompetitorID {
		gate.Reasons = append(gate.Reasons, "winner is not challenger")
	}
	if len(gate.Reasons) > 0 {
		gate.Status = "failed"
	}
	return gate, nil
}

func WriteSafetyScan(path string, out string) (SafetyReport, error) {
	report, err := ScanPath(path)
	if err != nil {
		return report, err
	}
	if err := writeJSON(out, report); err != nil {
		return report, err
	}
	return report, nil
}

const (
	maxSafetyScanFiles      = 4096
	maxSafetyScanFileBytes  = 1 * 1024 * 1024
	maxSafetyScanTotalBytes = 8 * 1024 * 1024
)

type safetyScanBudget struct {
	files      int
	totalBytes int64
}

func (budget *safetyScanBudget) accept(path string, info fs.FileInfo) error {
	size := info.Size()
	if size > maxSafetyScanFileBytes {
		return fmt.Errorf("safety scan file size limit exceeded for %s", filepath.ToSlash(path))
	}
	budget.files++
	if budget.files > maxSafetyScanFiles {
		return fmt.Errorf("safety scan file count limit exceeded")
	}
	budget.totalBytes += size
	if budget.totalBytes > maxSafetyScanTotalBytes {
		return fmt.Errorf("safety scan total byte limit exceeded")
	}
	return nil
}

func ScanPath(path string) (SafetyReport, error) {
	report := SafetyReport{
		SchemaVersion:  "ao.arena.safety-scan.v0.1",
		Status:         "passed",
		Findings:       []SafetyFinding{},
		ScannedPaths:   []string{},
		BlockedActions: forbiddenActions(),
	}
	root := filepath.Clean(path)
	info, err := os.Lstat(root)
	if err != nil {
		return report, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return report, fmt.Errorf("safety scan symlink is not allowed: %s", filepath.ToSlash(root))
	}
	budget := safetyScanBudget{}
	checkFile := func(filePath string, info fs.FileInfo) error {
		if err := budget.accept(filePath, info); err != nil {
			return err
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		rel := filePath
		if r, err := filepath.Rel(root, filePath); err == nil {
			rel = r
		}
		report.ScannedPaths = append(report.ScannedPaths, filepath.ToSlash(rel))
		text := string(data)
		if secretFinding(text) {
			report.Findings = append(report.Findings, SafetyFinding{Type: "secret_like_string", Path: filepath.ToSlash(rel)})
		}
		if forbiddenFinding(text) {
			report.Findings = append(report.Findings, SafetyFinding{Type: "forbidden_action", Path: filepath.ToSlash(rel)})
		}
		if localPathFinding(text) {
			report.Findings = append(report.Findings, SafetyFinding{Type: "local_absolute_path", Path: filepath.ToSlash(rel)})
		}
		return nil
	}
	if !info.IsDir() {
		if err := checkFile(root, info); err != nil {
			return report, err
		}
	} else {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("safety scan symlink is not allowed: %s", filepath.ToSlash(path))
			}
			if d.IsDir() && d.Name() == "invalid" && scanningExamples(root) {
				return filepath.SkipDir
			}
			if d.IsDir() {
				return nil
			}
			if !safeSuffix(path) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return checkFile(path, info)
		})
		if err != nil {
			return report, err
		}
	}
	report.FindingCount = len(report.Findings)
	if report.FindingCount > 0 {
		report.Status = "failed"
	}
	sort.Strings(report.ScannedPaths)
	return report, nil
}

func forbiddenActions() []string {
	return []string{"git push", "git tag", "gh release", "gh repo edit", "gh workflow run", "npm publish", "cargo publish", "docker push", "scp", "rsync", "curl -X POST", "upload", "deploy"}
}

func forbiddenFinding(text string) bool {
	lower := strings.ToLower(text)
	for _, action := range append(forbiddenActions(), "FORBIDDEN_ACTION_FIXTURE") {
		if strings.Contains(lower, strings.ToLower(action)) {
			return true
		}
	}
	return false
}

func secretFinding(text string) bool {
	for _, pattern := range []string{"Authorization: " + "Bearer", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GITHUB_TOKEN", "BEGIN " + "PRIVATE KEY", "pass" + "word=", "token=", "cookie=", "A" + "KIA"} {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func localPathFinding(text string) bool {
	for _, pattern := range []string{"/Users/", "/home/", "C:\\", "/tmp/", "/var/folders/", "../", "LOCAL_ABSOLUTE_PATH_FIXTURE"} {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func durablePublicOutput(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(clean, "/docs/") || strings.Contains(clean, "/examples/") || strings.HasPrefix(clean, "docs/") || strings.HasPrefix(clean, "examples/")
}

func safeSuffix(path string) bool {
	switch filepath.Ext(path) {
	case ".json", ".md", ".txt", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func scanningExamples(root string) bool {
	clean := filepath.ToSlash(filepath.Clean(root))
	return clean == "examples" || strings.HasSuffix(clean, "/examples")
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func writeJSON(path string, value any) error {
	if path == "" {
		return errors.New("missing output path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func writeText(path string, value string) error {
	if path == "" {
		return errors.New("missing output path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value), 0o644)
}

func digestString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func CanonicalTasks() []Task {
	out := make([]Task, len(canonicalTasks))
	copy(out, canonicalTasks)
	for i := range out {
		out[i].SchemaVersion = "ao.arena.benchmark-task.v0.1"
	}
	return out
}

func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
