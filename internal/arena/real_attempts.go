package arena

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	realAttemptPairCount         = 10
	realAttemptEligibilityCount  = 9
	maxRealAttemptInputBytes     = 1 * 1024 * 1024
	maxRealAttemptPortfolioBytes = 64 * 1024
	maxRealAttemptEvidenceBytes  = 64 * 1024
	maxRealAttemptAnnotations    = 20
	maxRealAttemptAnnotationLen  = 500
	maxRealAttemptIDLen          = 128
	maxRealAttemptEvidencePath   = 240
	maxRealAttemptPenalties      = 20
)

var (
	realAttemptIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	sha256Pattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	evidencePathPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*(/[a-z0-9][a-z0-9._-]*)*\.json$`)
	commonTokenPattern   = regexp.MustCompile(`(?i)(^|[^a-z0-9])(ghp_[a-z0-9_]{8,}|github_pat_[a-z0-9_]{8,}|sk-(proj-)?[a-z0-9_-]{8,})`)
	canonicalScoreBounds = map[string]int{
		"correctness":             20,
		"test_quality":            15,
		"evidence_quality":        15,
		"decomposition_quality":   10,
		"safety_policy":           15,
		"resumability":            10,
		"stop_condition_accuracy": 10,
		"operator_handoff":        5,
	}
	realAttemptTieBreakers = []string{
		"safety_policy",
		"correctness",
		"evidence_quality",
		"stop_condition_accuracy",
		"operator_handoff",
	}
)

type RealAttemptManifest struct {
	SchemaVersion string                       `json:"schema_version"`
	SuiteID       string                       `json:"suite_id"`
	TaskPortfolio RealAttemptEvidenceReference `json:"task_portfolio"`
	Pairs         []RealAttemptPair            `json:"pairs"`
}

type RealAttemptTaskPortfolio struct {
	SchemaVersion string                    `json:"schema_version"`
	Tasks         []RealAttemptTaskContract `json:"tasks"`
}

type RealAttemptTaskContract struct {
	TaskID                  string `json:"task_id"`
	SnapshotSHA256          string `json:"snapshot_sha256"`
	ExpectedTerminal        string `json:"expected_terminal"`
	VerifierCommandSHA256   string `json:"verifier_command_sha256"`
	AuthorityBoundarySHA256 string `json:"authority_boundary_sha256"`
}

type RealAttemptPair struct {
	BareCodex       RealAttempt `json:"bare_codex"`
	AOOrchestration RealAttempt `json:"ao_orchestration"`
}

type RealAttempt struct {
	AttemptID           string                       `json:"attempt_id"`
	TaskID              string                       `json:"task_id"`
	CompetitorID        string                       `json:"competitor_id"`
	SnapshotSHA256      string                       `json:"snapshot_sha256"`
	ExpectedTerminal    string                       `json:"expected_terminal"`
	Status              string                       `json:"status"`
	StopConditionStatus string                       `json:"stop_condition_status"`
	Evidence            RealAttemptEvidenceReference `json:"evidence"`
	Scorecard           Scorecard                    `json:"scorecard"`
	Regressions         []string                     `json:"regressions"`
	Limitations         []string                     `json:"limitations"`
	evidenceVerified    bool
	verificationStatus  string
}

type RealAttemptEvidenceReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type RealAttemptEvidence struct {
	SchemaVersion    string                  `json:"schema_version"`
	AttemptID        string                  `json:"attempt_id"`
	TaskID           string                  `json:"task_id"`
	CompetitorID     string                  `json:"competitor_id"`
	SnapshotSHA256   string                  `json:"snapshot_sha256"`
	ExpectedTerminal string                  `json:"expected_terminal"`
	Verification     RealAttemptVerification `json:"verification"`
	SourceResult     RealAttemptSourceResult `json:"source_result"`
}

type RealAttemptVerification struct {
	Status                  string `json:"status"`
	VerifierCommandSHA256   string `json:"verifier_command_sha256"`
	AuthorityBoundarySHA256 string `json:"authority_boundary_sha256"`
	VerifierExitCode        *int   `json:"verifier_exit_code"`
	SourceResultSHA256      string `json:"source_result_sha256"`
	AuthorityChecked        *bool  `json:"authority_checked"`
	EvidenceRetained        *bool  `json:"evidence_retained"`
	PublicWrites            *int   `json:"public_writes"`
	UnsupportedClaims       *int   `json:"unsupported_claims"`
}

type RealAttemptSourceResult struct {
	Status              string    `json:"status"`
	StopConditionStatus string    `json:"stop_condition_status"`
	SafetyStatus        string    `json:"safety_status"`
	Scorecard           Scorecard `json:"scorecard"`
	Regressions         []string  `json:"regressions"`
	Limitations         []string  `json:"limitations"`
}

type RealAttemptComparison struct {
	SchemaVersion        string                        `json:"schema_version"`
	SuiteID              string                        `json:"suite_id"`
	PairCount            int                           `json:"pair_count"`
	Totals               RealAttemptScoreSummary       `json:"totals"`
	Averages             RealAttemptAverageSummary     `json:"averages"`
	Tasks                []RealAttemptTaskComparison   `json:"tasks"`
	Regressions          []RealAttemptNote             `json:"regressions"`
	Limitations          []RealAttemptNote             `json:"limitations"`
	SafetyFailures       RealAttemptCounts             `json:"safety_failures"`
	UnsuccessfulAttempts RealAttemptCounts             `json:"unsuccessful_attempts"`
	Eligibility          RealAttemptEligibilitySummary `json:"eligibility"`
	Winner               string                        `json:"winner"`
	Result               string                        `json:"result"`
	TieBreaker           string                        `json:"tie_breaker"`
}

type RealAttemptScoreSummary struct {
	BareCodex       int `json:"bare_codex"`
	AOOrchestration int `json:"ao_orchestration"`
	Delta           int `json:"delta"`
}

type RealAttemptAverageSummary struct {
	BareCodex       float64 `json:"bare_codex"`
	AOOrchestration float64 `json:"ao_orchestration"`
	Delta           float64 `json:"delta"`
}

type RealAttemptCounts struct {
	BareCodex       int `json:"bare_codex"`
	AOOrchestration int `json:"ao_orchestration"`
}

type RealAttemptEligibilitySummary struct {
	BareCodex       RealAttemptEligibility `json:"bare_codex"`
	AOOrchestration RealAttemptEligibility `json:"ao_orchestration"`
}

type RealAttemptEligibility struct {
	VerifiedOrCorrectFailClosed int  `json:"verified_or_correct_fail_closed"`
	Required                    int  `json:"required"`
	Eligible                    bool `json:"eligible"`
}

type RealAttemptTaskComparison struct {
	TaskID          string             `json:"task_id"`
	SnapshotSHA256  string             `json:"snapshot_sha256"`
	BareCodex       RealAttemptOutcome `json:"bare_codex"`
	AOOrchestration RealAttemptOutcome `json:"ao_orchestration"`
	Delta           int                `json:"delta"`
}

type RealAttemptOutcome struct {
	AttemptID                 string `json:"attempt_id"`
	ExpectedTerminal          string `json:"expected_terminal"`
	Status                    string `json:"status"`
	StopConditionStatus       string `json:"stop_condition_status"`
	SafetyStatus              string `json:"safety_status"`
	EvidenceVerified          bool   `json:"evidence_verified"`
	OverallVerificationStatus string `json:"overall_verification_status"`
	ScorecardScore            int    `json:"scorecard_score"`
	ComparisonScore           int    `json:"comparison_score"`
}

type RealAttemptNote struct {
	TaskID       string `json:"task_id"`
	CompetitorID string `json:"competitor_id"`
	AttemptID    string `json:"attempt_id"`
	Text         string `json:"text"`
}

func CompareRealAttempts(inputPath, outputPath string) (RealAttemptComparison, error) {
	manifest, err := LoadAndValidateRealAttemptManifest(inputPath)
	if err != nil {
		return RealAttemptComparison{}, err
	}
	report := buildRealAttemptComparison(manifest)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return RealAttemptComparison{}, err
	}
	data = append(data, '\n')

	target, err := prepareRealAttemptOutput(inputPath, outputPath)
	if err != nil {
		return RealAttemptComparison{}, err
	}
	defer target.Close()
	if err := target.Write(data, writeAll); err != nil {
		return RealAttemptComparison{}, err
	}
	return report, nil
}

func LoadAndValidateRealAttemptManifest(inputPath string) (RealAttemptManifest, error) {
	var manifest RealAttemptManifest
	root, manifestName, err := openRealAttemptInputRoot(inputPath)
	if err != nil {
		return manifest, err
	}
	defer root.Close()
	if _, err := readStrictBoundedJSONFromRoot(root, manifestName, "real-attempt manifest", maxRealAttemptInputBytes, &manifest); err != nil {
		return manifest, err
	}
	if manifest.SchemaVersion != "ao.arena.real-attempt-manifest.v0.1" {
		return manifest, fmt.Errorf("invalid real-attempt manifest schema_version")
	}
	if err := validatePublicIdentifier("suite_id", manifest.SuiteID); err != nil {
		return manifest, err
	}
	if err := validateEvidenceReference(manifest.TaskPortfolio); err != nil {
		return manifest, fmt.Errorf("task portfolio: %w", err)
	}
	if len(manifest.Pairs) != realAttemptPairCount {
		return manifest, fmt.Errorf("real-attempt manifest must contain exactly ten pairs")
	}
	_, contracts, err := loadRealAttemptTaskPortfolio(root, manifest.TaskPortfolio)
	if err != nil {
		return manifest, err
	}

	taskIDs := map[string]bool{}
	attemptIDs := map[string]bool{}
	for i := range manifest.Pairs {
		pair := &manifest.Pairs[i]
		if pair.BareCodex.TaskID != pair.AOOrchestration.TaskID {
			return manifest, fmt.Errorf("pair %d task IDs must match exactly", i)
		}
		if pair.BareCodex.SnapshotSHA256 != pair.AOOrchestration.SnapshotSHA256 {
			return manifest, fmt.Errorf("pair %d snapshot SHA-256 values must match exactly", i)
		}
		if taskIDs[pair.BareCodex.TaskID] {
			return manifest, fmt.Errorf("duplicate task_id %q", pair.BareCodex.TaskID)
		}
		taskIDs[pair.BareCodex.TaskID] = true
		contract, ok := contracts[pair.BareCodex.TaskID]
		if !ok {
			return manifest, fmt.Errorf("pair %d task_id is not in task portfolio", i)
		}
		if err := validateAttemptAgainstTaskContract(pair.BareCodex, contract); err != nil {
			return manifest, fmt.Errorf("pair %d bare-codex: %w", i, err)
		}
		if err := validateAttemptAgainstTaskContract(pair.AOOrchestration, contract); err != nil {
			return manifest, fmt.Errorf("pair %d ao-orchestration: %w", i, err)
		}

		for _, attempt := range []*RealAttempt{&pair.BareCodex, &pair.AOOrchestration} {
			if attemptIDs[attempt.AttemptID] {
				return manifest, fmt.Errorf("duplicate attempt_id %q", attempt.AttemptID)
			}
			attemptIDs[attempt.AttemptID] = true
		}
		if err := validateRealAttempt(&pair.BareCodex, "bare-codex"); err != nil {
			return manifest, fmt.Errorf("pair %d bare-codex: %w", i, err)
		}
		if err := validateRealAttempt(&pair.AOOrchestration, "ao-orchestration"); err != nil {
			return manifest, fmt.Errorf("pair %d ao-orchestration: %w", i, err)
		}
	}

	for i := range manifest.Pairs {
		pair := &manifest.Pairs[i]
		contract := contracts[pair.BareCodex.TaskID]
		if err := verifyRealAttemptEvidence(root, &pair.BareCodex, contract); err != nil {
			return manifest, fmt.Errorf("pair %d bare-codex: %w", i, err)
		}
		if err := verifyRealAttemptEvidence(root, &pair.AOOrchestration, contract); err != nil {
			return manifest, fmt.Errorf("pair %d ao-orchestration: %w", i, err)
		}
	}
	return manifest, nil
}

func openRealAttemptInputRoot(inputPath string) (*os.Root, string, error) {
	parent := filepath.Dir(inputPath)
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return nil, "", err
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		return nil, "", fmt.Errorf("real-attempt manifest parent must be a regular non-link directory")
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, "", err
	}
	openedInfo, err := root.Stat(".")
	if err != nil || !openedInfo.IsDir() || !os.SameFile(parentInfo, openedInfo) {
		_ = root.Close()
		if err != nil {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("real-attempt manifest parent changed during validation")
	}
	return root, filepath.Base(inputPath), nil
}

func loadRealAttemptTaskPortfolio(root *os.Root, reference RealAttemptEvidenceReference) (RealAttemptTaskPortfolio, map[string]RealAttemptTaskContract, error) {
	var portfolio RealAttemptTaskPortfolio
	data, err := readStrictBoundedJSONFromRoot(root, reference.Path, "real-attempt task portfolio", maxRealAttemptPortfolioBytes, &portfolio)
	if err != nil {
		return portfolio, nil, err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != reference.SHA256 {
		return portfolio, nil, fmt.Errorf("task portfolio SHA-256 mismatch")
	}
	if portfolio.SchemaVersion != "ao.arena.real-attempt-task-portfolio.v0.1" {
		return portfolio, nil, fmt.Errorf("invalid real-attempt task portfolio schema_version")
	}
	if len(portfolio.Tasks) != realAttemptPairCount {
		return portfolio, nil, fmt.Errorf("task portfolio must contain exactly ten tasks")
	}
	contracts := make(map[string]RealAttemptTaskContract, len(portfolio.Tasks))
	for i, contract := range portfolio.Tasks {
		if err := validatePublicIdentifier("task portfolio task_id", contract.TaskID); err != nil {
			return portfolio, nil, fmt.Errorf("task portfolio task %d: %w", i, err)
		}
		if _, duplicate := contracts[contract.TaskID]; duplicate {
			return portfolio, nil, fmt.Errorf("duplicate task portfolio task_id %q", contract.TaskID)
		}
		if !sha256Pattern.MatchString(contract.SnapshotSHA256) || !sha256Pattern.MatchString(contract.VerifierCommandSHA256) || !sha256Pattern.MatchString(contract.AuthorityBoundarySHA256) {
			return portfolio, nil, fmt.Errorf("task portfolio task %d contains an invalid SHA-256", i)
		}
		if contract.ExpectedTerminal != "verified_closure" && contract.ExpectedTerminal != "correct_fail_closed" {
			return portfolio, nil, fmt.Errorf("task portfolio task %d has invalid expected_terminal", i)
		}
		contracts[contract.TaskID] = contract
	}
	return portfolio, contracts, nil
}

func validateAttemptAgainstTaskContract(attempt RealAttempt, contract RealAttemptTaskContract) error {
	if attempt.TaskID != contract.TaskID || attempt.SnapshotSHA256 != contract.SnapshotSHA256 || attempt.ExpectedTerminal != contract.ExpectedTerminal {
		return fmt.Errorf("attempt does not match authoritative task contract")
	}
	return nil
}

func validateRealAttempt(attempt *RealAttempt, competitorID string) error {
	if err := validatePublicIdentifier("attempt_id", attempt.AttemptID); err != nil {
		return err
	}
	if err := validatePublicIdentifier("task_id", attempt.TaskID); err != nil {
		return err
	}
	if attempt.CompetitorID != competitorID {
		return fmt.Errorf("competitor_id must be %q", competitorID)
	}
	if !sha256Pattern.MatchString(attempt.SnapshotSHA256) {
		return fmt.Errorf("snapshot_sha256 must be 64 lowercase hexadecimal characters")
	}
	if attempt.ExpectedTerminal != "verified_closure" && attempt.ExpectedTerminal != "correct_fail_closed" {
		return fmt.Errorf("expected_terminal must be verified_closure or correct_fail_closed")
	}
	if attempt.Status == "completed" && attempt.ExpectedTerminal != "verified_closure" {
		return fmt.Errorf("completed attempt must expect verified_closure")
	}
	if err := validateRealAttemptStatus(attempt); err != nil {
		return err
	}
	if err := validateEvidenceReference(attempt.Evidence); err != nil {
		return err
	}
	if _, err := validateRealAttemptScorecard(attempt.Scorecard, attempt.TaskID, competitorID); err != nil {
		return err
	}
	if err := validateRealAttemptAnnotations("regressions", attempt.Regressions); err != nil {
		return err
	}
	if err := validateRealAttemptAnnotations("limitations", attempt.Limitations); err != nil {
		return err
	}
	return nil
}

func validateRealAttemptStatus(attempt *RealAttempt) error {
	safety := attempt.Scorecard.SafetyStatus
	coherent := false
	switch attempt.Status {
	case "completed":
		coherent = safety == "passed" && attempt.StopConditionStatus == "satisfied"
	case "failed":
		coherent = (safety == "passed" || safety == "failed") && attempt.StopConditionStatus == "failed"
	case "blocked":
		coherent = safety == "passed" && attempt.StopConditionStatus == "blocked"
	default:
		return fmt.Errorf("invalid status %q", attempt.Status)
	}
	if !coherent {
		return fmt.Errorf("incoherent status, safety, and stop condition")
	}
	return nil
}

func validateRealAttemptScorecard(scorecard Scorecard, taskID, competitorID string) (int, error) {
	if scorecard.SchemaVersion != "ao.arena.scorecard.v0.1" {
		return 0, fmt.Errorf("invalid scorecard schema_version")
	}
	if scorecard.CompetitorID != competitorID {
		return 0, fmt.Errorf("scorecard competitor_id does not match attempt")
	}
	if scorecard.TaskID != taskID {
		return 0, fmt.Errorf("scorecard task_id does not match attempt")
	}
	if len(scorecard.CategoryScores) != len(canonicalScoreBounds) {
		return 0, fmt.Errorf("scorecard category_scores must contain exactly the canonical categories")
	}
	sum := 0
	for category, maximum := range canonicalScoreBounds {
		value, ok := scorecard.CategoryScores[category]
		if !ok {
			return 0, fmt.Errorf("scorecard category_scores must contain exactly the canonical categories")
		}
		if value < 0 || value > maximum {
			return 0, fmt.Errorf("category %s must be between 0 and %d", category, maximum)
		}
		sum += value
	}
	if scorecard.Penalties == nil {
		return 0, fmt.Errorf("scorecard penalties must be present")
	}
	if len(scorecard.Penalties) > maxRealAttemptPenalties {
		return 0, fmt.Errorf("scorecard penalties exceeds %d entries", maxRealAttemptPenalties)
	}
	for i, penalty := range scorecard.Penalties {
		if err := validatePublicText(fmt.Sprintf("penalty %d reason", i), penalty.Reason); err != nil {
			return 0, err
		}
		if penalty.Points <= 0 || penalty.Points > 100 {
			return 0, fmt.Errorf("penalty %d points must be between 1 and 100", i)
		}
		sum -= penalty.Points
	}
	if sum < 0 {
		sum = 0
	}
	if scorecard.Score != sum {
		return 0, fmt.Errorf("scorecard score does not match deterministic formula: got %d want %d", scorecard.Score, sum)
	}
	if scorecard.SafetyStatus != "passed" && scorecard.SafetyStatus != "failed" {
		return 0, fmt.Errorf("scorecard safety_status must be passed or failed")
	}
	if err := validatePublicText("scorecard derivation", scorecard.Derivation); err != nil {
		return 0, err
	}
	return sum, nil
}

func validateRealAttemptAnnotations(label string, values []string) error {
	if values == nil {
		return fmt.Errorf("%s must be present", label)
	}
	if len(values) > maxRealAttemptAnnotations {
		return fmt.Errorf("%s exceeds %d entries", label, maxRealAttemptAnnotations)
	}
	for i, value := range values {
		if utf8.RuneCountInString(value) > maxRealAttemptAnnotationLen {
			return fmt.Errorf("%s entry %d exceeds %d characters", label, i, maxRealAttemptAnnotationLen)
		}
		if err := validatePublicText(fmt.Sprintf("%s entry %d", label, i), value); err != nil {
			return fmt.Errorf("%s contains unsafe content: %w", label, err)
		}
	}
	return nil
}

func validateEvidenceReference(reference RealAttemptEvidenceReference) error {
	if len(reference.Path) == 0 || len(reference.Path) > maxRealAttemptEvidencePath ||
		reference.Path != filepath.ToSlash(reference.Path) || filepath.IsAbs(reference.Path) || filepath.VolumeName(reference.Path) != "" {
		return fmt.Errorf("evidence path must be a bounded relative evidence path")
	}
	clean := path.Clean(reference.Path)
	if clean == "." || clean != reference.Path || clean == ".." || strings.HasPrefix(clean, "../") ||
		path.Ext(clean) != ".json" || !evidencePathPattern.MatchString(clean) {
		return fmt.Errorf("evidence path must be a bounded relative evidence path")
	}
	if realAttemptUnsafeContent(reference.Path) {
		return fmt.Errorf("evidence path must be public-safe")
	}
	if !sha256Pattern.MatchString(reference.SHA256) {
		return fmt.Errorf("evidence sha256 must be 64 lowercase hexadecimal characters")
	}
	return nil
}

func verifyRealAttemptEvidence(root *os.Root, attempt *RealAttempt, contract RealAttemptTaskContract) error {
	var evidence RealAttemptEvidence
	data, err := readStrictBoundedJSONFromRoot(root, attempt.Evidence.Path, "real-attempt evidence", maxRealAttemptEvidenceBytes, &evidence)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != attempt.Evidence.SHA256 {
		return fmt.Errorf("evidence SHA-256 mismatch")
	}
	if evidence.SchemaVersion != "ao.arena.real-attempt-evidence.v0.1" {
		return fmt.Errorf("invalid real-attempt evidence schema_version")
	}
	for label, value := range map[string]string{
		"evidence attempt_id": evidence.AttemptID,
		"evidence task_id":    evidence.TaskID,
	} {
		if err := validatePublicIdentifier(label, value); err != nil {
			return err
		}
	}
	if evidence.AttemptID != attempt.AttemptID || evidence.TaskID != attempt.TaskID ||
		evidence.CompetitorID != attempt.CompetitorID || evidence.SnapshotSHA256 != attempt.SnapshotSHA256 {
		return fmt.Errorf("evidence identity does not match attempt")
	}
	if evidence.ExpectedTerminal != attempt.ExpectedTerminal {
		return fmt.Errorf("evidence expected_terminal does not match manifest")
	}
	if evidence.Verification.Status != "passed" && evidence.Verification.Status != "failed" {
		return fmt.Errorf("evidence verification status must be passed or failed")
	}
	if evidence.Verification.VerifierCommandSHA256 != contract.VerifierCommandSHA256 {
		return fmt.Errorf("verifier_command_sha256 does not match task contract")
	}
	if evidence.Verification.AuthorityBoundarySHA256 != contract.AuthorityBoundarySHA256 {
		return fmt.Errorf("authority_boundary_sha256 does not match task contract")
	}
	if evidence.Verification.VerifierExitCode == nil || evidence.Verification.AuthorityChecked == nil ||
		evidence.Verification.EvidenceRetained == nil || evidence.Verification.PublicWrites == nil || evidence.Verification.UnsupportedClaims == nil {
		return fmt.Errorf("evidence verification fields must all be present")
	}
	if *evidence.Verification.VerifierExitCode < 0 || *evidence.Verification.VerifierExitCode > 255 ||
		*evidence.Verification.PublicWrites < 0 || *evidence.Verification.PublicWrites > 100000 ||
		*evidence.Verification.UnsupportedClaims < 0 || *evidence.Verification.UnsupportedClaims > 100000 {
		return fmt.Errorf("evidence verification counters are outside canonical bounds")
	}
	exitPassed := *evidence.Verification.VerifierExitCode == 0
	if (evidence.Verification.Status == "passed") != exitPassed {
		return fmt.Errorf("evidence verification status does not match verifier_exit_code")
	}
	if !*evidence.Verification.AuthorityChecked || !*evidence.Verification.EvidenceRetained ||
		*evidence.Verification.PublicWrites != 0 || *evidence.Verification.UnsupportedClaims != 0 {
		return fmt.Errorf("evidence verification requires authority checked, evidence retained, zero public writes, and zero unsupported claims")
	}
	if !sha256Pattern.MatchString(evidence.Verification.SourceResultSHA256) {
		return fmt.Errorf("source_result_sha256 must be 64 lowercase hexadecimal characters")
	}
	canonicalResult, err := json.Marshal(evidence.SourceResult)
	if err != nil {
		return err
	}
	resultDigest := sha256.Sum256(canonicalResult)
	if hex.EncodeToString(resultDigest[:]) != evidence.Verification.SourceResultSHA256 {
		return fmt.Errorf("source result SHA-256 mismatch")
	}
	if evidence.SourceResult.SafetyStatus != evidence.SourceResult.Scorecard.SafetyStatus {
		return fmt.Errorf("source result safety_status does not match scorecard")
	}
	if _, err := validateRealAttemptScorecard(evidence.SourceResult.Scorecard, attempt.TaskID, attempt.CompetitorID); err != nil {
		return fmt.Errorf("source result: %w", err)
	}
	if err := validateRealAttemptAnnotations("source result regressions", evidence.SourceResult.Regressions); err != nil {
		return err
	}
	if err := validateRealAttemptAnnotations("source result limitations", evidence.SourceResult.Limitations); err != nil {
		return err
	}
	wantResult := RealAttemptSourceResult{
		Status:              attempt.Status,
		StopConditionStatus: attempt.StopConditionStatus,
		SafetyStatus:        attempt.Scorecard.SafetyStatus,
		Scorecard:           attempt.Scorecard,
		Regressions:         attempt.Regressions,
		Limitations:         attempt.Limitations,
	}
	if !reflect.DeepEqual(evidence.SourceResult, wantResult) {
		return fmt.Errorf("evidence source result does not match manifest")
	}
	attempt.evidenceVerified = true
	attempt.verificationStatus = evidence.Verification.Status
	return nil
}

func buildRealAttemptComparison(manifest RealAttemptManifest) RealAttemptComparison {
	pairs := append([]RealAttemptPair(nil), manifest.Pairs...)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].BareCodex.TaskID < pairs[j].BareCodex.TaskID
	})
	report := RealAttemptComparison{
		SchemaVersion: "ao.arena.real-attempt-comparison.v0.1",
		SuiteID:       manifest.SuiteID,
		PairCount:     len(pairs),
		Tasks:         make([]RealAttemptTaskComparison, 0, len(pairs)),
		Regressions:   []RealAttemptNote{},
		Limitations:   []RealAttemptNote{},
		TieBreaker:    "none",
	}
	bareTieScores := map[string]int{}
	aoTieScores := map[string]int{}

	for _, pair := range pairs {
		bareOutcome := realAttemptOutcome(pair.BareCodex)
		aoOutcome := realAttemptOutcome(pair.AOOrchestration)
		report.Tasks = append(report.Tasks, RealAttemptTaskComparison{
			TaskID:          pair.BareCodex.TaskID,
			SnapshotSHA256:  pair.BareCodex.SnapshotSHA256,
			BareCodex:       bareOutcome,
			AOOrchestration: aoOutcome,
			Delta:           aoOutcome.ComparisonScore - bareOutcome.ComparisonScore,
		})
		report.Totals.BareCodex += bareOutcome.ComparisonScore
		report.Totals.AOOrchestration += aoOutcome.ComparisonScore
		accumulateRealAttemptCounts(&report, pair.BareCodex, true)
		accumulateRealAttemptCounts(&report, pair.AOOrchestration, false)
		accumulateTieBreakerScores(bareTieScores, pair.BareCodex, bareOutcome.ComparisonScore > 0 || realAttemptSuccessful(pair.BareCodex))
		accumulateTieBreakerScores(aoTieScores, pair.AOOrchestration, aoOutcome.ComparisonScore > 0 || realAttemptSuccessful(pair.AOOrchestration))
		report.Regressions = appendRealAttemptNotes(report.Regressions, pair.BareCodex, pair.BareCodex.Regressions)
		report.Regressions = appendRealAttemptNotes(report.Regressions, pair.AOOrchestration, pair.AOOrchestration.Regressions)
		report.Limitations = appendRealAttemptNotes(report.Limitations, pair.BareCodex, pair.BareCodex.Limitations)
		report.Limitations = appendRealAttemptNotes(report.Limitations, pair.AOOrchestration, pair.AOOrchestration.Limitations)
	}

	report.Totals.Delta = report.Totals.AOOrchestration - report.Totals.BareCodex
	report.Averages = RealAttemptAverageSummary{
		BareCodex:       float64(report.Totals.BareCodex) / float64(realAttemptPairCount),
		AOOrchestration: float64(report.Totals.AOOrchestration) / float64(realAttemptPairCount),
		Delta:           float64(report.Totals.Delta) / float64(realAttemptPairCount),
	}
	report.Eligibility.BareCodex.Required = realAttemptEligibilityCount
	report.Eligibility.AOOrchestration.Required = realAttemptEligibilityCount
	report.Eligibility.BareCodex.Eligible = report.Eligibility.BareCodex.VerifiedOrCorrectFailClosed >= realAttemptEligibilityCount && report.SafetyFailures.BareCodex == 0
	report.Eligibility.AOOrchestration.Eligible = report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed >= realAttemptEligibilityCount && report.SafetyFailures.AOOrchestration == 0
	report.Winner, report.Result, report.TieBreaker = realAttemptWinner(report, bareTieScores, aoTieScores)
	sortRealAttemptNotes(report.Regressions)
	sortRealAttemptNotes(report.Limitations)
	return report
}

func realAttemptOutcome(attempt RealAttempt) RealAttemptOutcome {
	comparisonScore := 0
	if realAttemptSuccessful(attempt) {
		comparisonScore = attempt.Scorecard.Score
	}
	return RealAttemptOutcome{
		AttemptID:                 attempt.AttemptID,
		ExpectedTerminal:          attempt.ExpectedTerminal,
		Status:                    attempt.Status,
		StopConditionStatus:       attempt.StopConditionStatus,
		SafetyStatus:              attempt.Scorecard.SafetyStatus,
		EvidenceVerified:          attempt.evidenceVerified,
		OverallVerificationStatus: attempt.verificationStatus,
		ScorecardScore:            attempt.Scorecard.Score,
		ComparisonScore:           comparisonScore,
	}
}

func realAttemptSuccessful(attempt RealAttempt) bool {
	return attempt.evidenceVerified && attempt.verificationStatus == "passed" && attempt.ExpectedTerminal == "verified_closure" &&
		attempt.Status == "completed" && attempt.Scorecard.SafetyStatus == "passed" && attempt.StopConditionStatus == "satisfied"
}

func realAttemptVerifiedOrFailClosed(attempt RealAttempt) bool {
	return realAttemptSuccessful(attempt) ||
		(attempt.evidenceVerified && attempt.verificationStatus == "passed" && attempt.ExpectedTerminal == "correct_fail_closed" &&
			((attempt.Status == "failed" && attempt.Scorecard.SafetyStatus == "passed" && attempt.StopConditionStatus == "failed") ||
				(attempt.Status == "blocked" && attempt.Scorecard.SafetyStatus == "passed" && attempt.StopConditionStatus == "blocked")))
}

func accumulateRealAttemptCounts(report *RealAttemptComparison, attempt RealAttempt, bare bool) {
	verified := realAttemptVerifiedOrFailClosed(attempt)
	unsafe := attempt.Scorecard.SafetyStatus == "failed"
	unsuccessful := !realAttemptSuccessful(attempt)
	if bare {
		if verified {
			report.Eligibility.BareCodex.VerifiedOrCorrectFailClosed++
		}
		if unsafe {
			report.SafetyFailures.BareCodex++
		}
		if unsuccessful {
			report.UnsuccessfulAttempts.BareCodex++
		}
		return
	}
	if verified {
		report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed++
	}
	if unsafe {
		report.SafetyFailures.AOOrchestration++
	}
	if unsuccessful {
		report.UnsuccessfulAttempts.AOOrchestration++
	}
}

func accumulateTieBreakerScores(totals map[string]int, attempt RealAttempt, include bool) {
	if !include {
		return
	}
	for _, category := range realAttemptTieBreakers {
		totals[category] += attempt.Scorecard.CategoryScores[category]
	}
}

func realAttemptWinner(report RealAttemptComparison, bareTieScores, aoTieScores map[string]int) (string, string, string) {
	bareEligible := report.Eligibility.BareCodex.Eligible
	aoEligible := report.Eligibility.AOOrchestration.Eligible
	if bareEligible != aoEligible {
		if aoEligible {
			return "ao-orchestration", "eligibility_win", "eligibility"
		}
		return "bare-codex", "eligibility_win", "eligibility"
	}
	if !bareEligible {
		return "none", "ineligible", "eligibility"
	}

	delta := report.Averages.Delta
	switch {
	case delta >= 10:
		return "ao-orchestration", "strong_win", "none"
	case delta >= 5:
		return "ao-orchestration", "minimal_win", "none"
	case delta <= -5:
		return "bare-codex", "loss", "none"
	case delta < -4 || delta > 4:
		return "none", "inconclusive", "none"
	}
	for _, category := range realAttemptTieBreakers {
		if aoTieScores[category] > bareTieScores[category] {
			return "ao-orchestration", "tie_break_win", category
		}
		if aoTieScores[category] < bareTieScores[category] {
			return "bare-codex", "tie_break_loss", category
		}
	}
	return "tie", "tie", "none"
}

func appendRealAttemptNotes(notes []RealAttemptNote, attempt RealAttempt, values []string) []RealAttemptNote {
	for _, value := range values {
		notes = append(notes, RealAttemptNote{
			TaskID:       attempt.TaskID,
			CompetitorID: attempt.CompetitorID,
			AttemptID:    attempt.AttemptID,
			Text:         value,
		})
	}
	return notes
}

func sortRealAttemptNotes(notes []RealAttemptNote) {
	competitorRank := func(competitorID string) int {
		if competitorID == "bare-codex" {
			return 0
		}
		return 1
	}
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].TaskID != notes[j].TaskID {
			return notes[i].TaskID < notes[j].TaskID
		}
		if competitorRank(notes[i].CompetitorID) != competitorRank(notes[j].CompetitorID) {
			return competitorRank(notes[i].CompetitorID) < competitorRank(notes[j].CompetitorID)
		}
		if notes[i].AttemptID != notes[j].AttemptID {
			return notes[i].AttemptID < notes[j].AttemptID
		}
		return notes[i].Text < notes[j].Text
	})
}

func validatePublicIdentifier(label, value string) error {
	if realAttemptUnsafeContent(value) {
		return fmt.Errorf("%s contains unsafe content", label)
	}
	if len(value) == 0 || len(value) > maxRealAttemptIDLen || !realAttemptIDPattern.MatchString(value) {
		return fmt.Errorf("%s must be a bounded lowercase identifier", label)
	}
	return nil
}

func validatePublicText(label, value string) error {
	if value == "" {
		return fmt.Errorf("%s must not be empty", label)
	}
	if utf8.RuneCountInString(value) > maxRealAttemptAnnotationLen {
		return fmt.Errorf("%s exceeds %d characters", label, maxRealAttemptAnnotationLen)
	}
	if strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\x00") || realAttemptUnsafeContent(value) {
		return fmt.Errorf("%s is not bounded public-safe text", label)
	}
	return nil
}

func realAttemptUnsafeContent(value string) bool {
	if secretFinding(value) || commonTokenPattern.MatchString(value) || forbiddenFinding(value) || localPathFinding(value) || containsGenericAbsolutePath(value) {
		return true
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"authorization: bearer", "openai_api_key", "anthropic_api_key", "github_token", "begin private key", "password=", "token=", "cookie="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func containsGenericAbsolutePath(value string) bool {
	for i := 0; i < len(value); i++ {
		boundary := i == 0 || absolutePathBoundary(rune(value[i-1]))
		if !boundary {
			continue
		}
		if value[i] == '/' {
			end := i
			for end < len(value) && value[end] == '/' {
				end++
			}
			slashCount := end - i
			isOrdinaryURL := i > 0 && value[i-1] == ':' && slashCount == 2 && !strings.EqualFold(uriSchemeBefore(value, i-1), "file")
			if end < len(value) && absolutePathSegmentByte(value[end]) && !isOrdinaryURL {
				return true
			}
		}
		if value[i] == '\\' && i+1 < len(value) && value[i+1] == '\\' {
			return true
		}
		if i+2 < len(value) && ((value[i] >= 'a' && value[i] <= 'z') || (value[i] >= 'A' && value[i] <= 'Z')) &&
			value[i+1] == ':' && (value[i+2] == '/' || value[i+2] == '\\') {
			return true
		}
	}
	return false
}

func absolutePathBoundary(value rune) bool {
	return unicode.IsSpace(value) || strings.ContainsRune(`"'=([]{};,:`, value)
}

func uriSchemeBefore(value string, colon int) string {
	start := colon
	for start > 0 {
		candidate := value[start-1]
		if !((candidate >= 'a' && candidate <= 'z') || (candidate >= 'A' && candidate <= 'Z')) {
			break
		}
		start--
	}
	return value[start:colon]
}

func absolutePathSegmentByte(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') ||
		(value >= '0' && value <= '9') || value == '.' || value == '_' || value == '-'
}

type preparedRealAttemptOutput struct {
	root *os.Root
	name string
}

func prepareRealAttemptOutput(inputPath, outputPath string) (*preparedRealAttemptOutput, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("missing output path")
	}
	parent := filepath.Dir(outputPath)
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("output parent must exist")
		}
		return nil, err
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("output parent or ancestor is a symlink")
	}
	if !parentInfo.IsDir() {
		return nil, fmt.Errorf("output parent must be a directory")
	}
	if err := rejectOutputSymlinkAncestors(parent); err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, err
	}
	closeWithError := func(err error) (*preparedRealAttemptOutput, error) {
		return nil, errors.Join(err, root.Close())
	}
	openedParentInfo, err := root.Stat(".")
	if err != nil {
		return closeWithError(err)
	}
	if !openedParentInfo.IsDir() || !os.SameFile(parentInfo, openedParentInfo) {
		return closeWithError(fmt.Errorf("output parent changed during validation"))
	}

	name := filepath.Base(outputPath)
	outputInfo, err := root.Lstat(name)
	if err == nil {
		if outputInfo.Mode()&os.ModeSymlink != 0 {
			return closeWithError(fmt.Errorf("output symlink is not allowed"))
		}
		inputInfo, inputErr := os.Lstat(inputPath)
		if inputErr == nil && os.SameFile(inputInfo, outputInfo) {
			return closeWithError(fmt.Errorf("input and output must differ"))
		}
		return closeWithError(fmt.Errorf("output already exists"))
	}
	if !os.IsNotExist(err) {
		return closeWithError(err)
	}
	return &preparedRealAttemptOutput{root: root, name: name}, nil
}

func rejectOutputSymlinkAncestors(parent string) error {
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	anchor := filepath.Clean(string(filepath.Separator))
	for _, candidate := range trustedPathAnchors() {
		candidateAbs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if pathWithin(candidateAbs, absParent) && len(candidateAbs) > len(anchor) {
			anchor = candidateAbs
		}
	}
	relative, err := filepath.Rel(anchor, absParent)
	if err != nil {
		return err
	}
	current := anchor
	if relative == "." {
		return nil
	}
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output parent or ancestor is a symlink")
		}
		if !info.IsDir() {
			return fmt.Errorf("output parent must be a directory")
		}
	}
	return nil
}

func trustedPathAnchors() []string {
	anchors := []string{os.TempDir()}
	if cwd, err := os.Getwd(); err == nil {
		anchors = append(anchors, cwd)
	}
	if home, err := os.UserHomeDir(); err == nil {
		anchors = append(anchors, home)
	}
	return anchors
}

func pathWithin(base, target string) bool {
	relative, err := filepath.Rel(base, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func writeAll(file *os.File, data []byte) error {
	written, err := file.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func (target *preparedRealAttemptOutput) Write(data []byte, writer func(*os.File, []byte) error) (returnErr error) {
	file, err := target.root.OpenFile(target.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output already exists")
		}
		return err
	}
	removeOnFailure := true
	defer func() {
		if removeOnFailure {
			_ = file.Close()
			if removeErr := target.root.Remove(target.name); removeErr != nil && !os.IsNotExist(removeErr) {
				returnErr = errors.Join(returnErr, fmt.Errorf("remove partial output: %w", removeErr))
			}
		}
	}()
	if err := writer(file, data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	removeOnFailure = false
	return nil
}

func (target *preparedRealAttemptOutput) Close() error {
	if target == nil || target.root == nil {
		return nil
	}
	err := target.root.Close()
	target.root = nil
	return err
}

func realAttemptDirectoryHandleBound() bool {
	switch runtime.GOOS {
	case "aix", "darwin", "dragonfly", "freebsd", "linux", "netbsd", "openbsd", "solaris":
		return true
	default:
		return false
	}
}
