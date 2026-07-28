package arena

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

const (
	autonomousRepairGovernanceEvaluationSchema = "ao.arena.autonomous-repair-governance-evaluation.v1"
	autonomousRepairGovernanceEvaluationID     = "stage5-autonomous-repair-github-governance-evaluation"
	autonomousRepairGovernanceResultsSHA256    = "71c5980167c31fd8bfae7595edd5ccb54491fbb41598992ec1108fb4ede99b07"
	autonomousRepairGovernanceEvaluationLimit  = 1 << 20
)

var lowercaseSHA256 = regexp.MustCompile(`^[0-9a-f]{64}$`)

type AutonomousRepairGovernanceEvaluation struct {
	SchemaVersion      string                               `json:"schema_version"`
	EvaluationID       string                               `json:"evaluation_id"`
	Status             string                               `json:"status"`
	Provenance         AutonomousRepairGovernanceProvenance `json:"provenance"`
	Subjects           AutonomousRepairGovernanceSubjects   `json:"subjects"`
	SafetyBoundary     AutonomousRepairGovernanceSafety     `json:"safety_boundary"`
	Results            []AutonomousRepairGovernanceResult   `json:"results"`
	ResultsSHA256      string                               `json:"results_sha256"`
	Metrics            AutonomousRepairGovernanceMetrics    `json:"metrics"`
	Coverage           AutonomousRepairGovernanceCoverage   `json:"coverage"`
	PromotionRequested bool                                 `json:"promotion_requested"`
	CanonicalDigest    string                               `json:"canonical_digest,omitempty"`
}

type AutonomousRepairGovernanceProvenance struct {
	Crucible AutonomousRepairGovernanceCrucibleProvenance `json:"crucible"`
}

type AutonomousRepairGovernanceCrucibleProvenance struct {
	Repository           string `json:"repository"`
	Commit               string `json:"commit"`
	SuitePath            string `json:"suite_path"`
	SuiteSHA256          string `json:"suite_sha256"`
	SchemaPath           string `json:"schema_path"`
	SchemaSHA256         string `json:"schema_sha256"`
	ImplementationPath   string `json:"implementation_path"`
	ImplementationSHA256 string `json:"implementation_sha256"`
	SuiteCanonicalDigest string `json:"suite_canonical_digest"`
}

type AutonomousRepairGovernanceSubjects struct {
	CovenantCommit string `json:"covenant_commit"`
	AO2Commit      string `json:"ao2_commit"`
}

type AutonomousRepairGovernanceSafety struct {
	Mode                  string `json:"mode"`
	LiveProviderUsed      bool   `json:"live_provider_used"`
	NetworkUsed           bool   `json:"network_used"`
	CredentialsUsed       bool   `json:"credentials_used"`
	GitHubMutation        bool   `json:"github_mutation"`
	SiblingRepoMutation   bool   `json:"sibling_repo_mutation"`
	ProviderCallPerformed bool   `json:"provider_call_performed"`
}

type AutonomousRepairGovernanceResult struct {
	CaseID        string `json:"case_id"`
	InputSHA256   string `json:"input_sha256"`
	Authorized    bool   `json:"authorized"`
	ReasonCode    string `json:"reason_code"`
	WriteCount    int    `json:"write_count"`
	TerminalState string `json:"terminal_state"`
	Conformant    bool   `json:"conformant"`
}

type AutonomousRepairGovernanceMetrics struct {
	CaseCount                int `json:"case_count"`
	AuthorizedCount          int `json:"authorized_count"`
	DeniedCount              int `json:"denied_count"`
	TotalWriteCount          int `json:"total_write_count"`
	MaxWriteCount            int `json:"max_write_count"`
	UniqueReasonCount        int `json:"unique_reason_count"`
	AuthorizedZeroWriteCount int `json:"authorized_zero_write_count"`
	DeniedAfterWriteCount    int `json:"denied_after_write_count"`
	ConformanceFailures      int `json:"conformance_failures"`
	DuplicateCaseIDs         int `json:"duplicate_case_ids"`
	DuplicateInputDigests    int `json:"duplicate_input_digests"`
}

type AutonomousRepairGovernanceCoverage struct {
	SoleControlOptIn            bool `json:"sole_control_opt_in"`
	TeamHumanCodeownerExactHead bool `json:"team_human_codeowner_exact_head"`
	AutomatedReviewerDenied     bool `json:"automated_reviewer_denied"`
	ExternalUnknownDraftOnly    bool `json:"external_unknown_draft_only"`
	ClassSensitiveReadyPR       bool `json:"class_sensitive_ready_pr"`
	ExactForkBranchPrerequisite bool `json:"exact_fork_branch_prerequisite"`
	DuplicateDraftIdempotence   bool `json:"duplicate_draft_idempotence"`
	ConflictReread              bool `json:"conflict_reread"`
	PostWriteReadback           bool `json:"post_write_readback"`
	PerWriteExpiry              bool `json:"per_write_expiry"`
	PermanentWriteDenial        bool `json:"permanent_write_denial"`
	ProtectedChecksDigest       bool `json:"protected_checks_digest"`
}

var expectedAutonomousRepairGovernanceMetrics = AutonomousRepairGovernanceMetrics{
	CaseCount: 47, AuthorizedCount: 10, DeniedCount: 37, TotalWriteCount: 16,
	MaxWriteCount: 2, UniqueReasonCount: 32, AuthorizedZeroWriteCount: 2,
	DeniedAfterWriteCount: 7,
}

var expectedAutonomousRepairGovernanceCoverage = AutonomousRepairGovernanceCoverage{
	SoleControlOptIn: true, TeamHumanCodeownerExactHead: true, AutomatedReviewerDenied: true,
	ExternalUnknownDraftOnly: true, ClassSensitiveReadyPR: true,
	ExactForkBranchPrerequisite: true, DuplicateDraftIdempotence: true,
	ConflictReread: true, PostWriteReadback: true, PerWriteExpiry: true,
	PermanentWriteDenial: true, ProtectedChecksDigest: true,
}

func LoadAndValidateAutonomousRepairGovernanceEvaluation(path string) (AutonomousRepairGovernanceEvaluation, error) {
	var report AutonomousRepairGovernanceEvaluation
	data, err := readStrictBoundedJSON(
		path,
		"autonomous repair governance evaluation",
		autonomousRepairGovernanceEvaluationLimit,
		&report,
	)
	if err != nil {
		return report, err
	}
	return DecodeAndValidateAutonomousRepairGovernanceEvaluation(data)
}

func DecodeAndValidateAutonomousRepairGovernanceEvaluation(data []byte) (AutonomousRepairGovernanceEvaluation, error) {
	var report AutonomousRepairGovernanceEvaluation
	if len(data) > autonomousRepairGovernanceEvaluationLimit {
		return report, fmt.Errorf("governance evaluation size limit exceeded")
	}
	if err := decodeStrictJSON(data, &report); err != nil {
		return report, fmt.Errorf("decode governance evaluation: %w", err)
	}
	if err := requireAutonomousRepairGovernanceFields(data); err != nil {
		return report, err
	}
	if err := validateAutonomousRepairGovernanceEvaluation(report); err != nil {
		return report, err
	}
	return report, nil
}

func EvaluateAutonomousRepairGovernanceResults(results []AutonomousRepairGovernanceResult) (AutonomousRepairGovernanceMetrics, error) {
	var metrics AutonomousRepairGovernanceMetrics
	caseIDs := make(map[string]bool, len(results))
	inputDigests := make(map[string]bool, len(results))
	reasons := make(map[string]bool, len(results))

	for index, result := range results {
		if result.CaseID == "" || result.ReasonCode == "" {
			return metrics, fmt.Errorf("result %d has an empty identifier or reason", index)
		}
		if !lowercaseSHA256.MatchString(result.InputSHA256) {
			return metrics, fmt.Errorf("result %q has an invalid input_sha256", result.CaseID)
		}
		if result.WriteCount < 0 || result.WriteCount > 2 {
			return metrics, fmt.Errorf("result %q write_count is outside 0..2", result.CaseID)
		}
		if result.Authorized && result.TerminalState != "authorized" {
			return metrics, fmt.Errorf("result %q authorized state is inconsistent", result.CaseID)
		}
		if !result.Authorized && result.TerminalState != "denied" {
			return metrics, fmt.Errorf("result %q denied state is inconsistent", result.CaseID)
		}

		metrics.CaseCount++
		metrics.TotalWriteCount += result.WriteCount
		if result.WriteCount > metrics.MaxWriteCount {
			metrics.MaxWriteCount = result.WriteCount
		}
		if result.Authorized {
			metrics.AuthorizedCount++
			if result.WriteCount == 0 {
				metrics.AuthorizedZeroWriteCount++
			}
		} else {
			metrics.DeniedCount++
			if result.WriteCount > 0 {
				metrics.DeniedAfterWriteCount++
			}
		}
		if !result.Conformant {
			metrics.ConformanceFailures++
		}
		if caseIDs[result.CaseID] {
			metrics.DuplicateCaseIDs++
		}
		caseIDs[result.CaseID] = true
		if inputDigests[result.InputSHA256] {
			metrics.DuplicateInputDigests++
		}
		inputDigests[result.InputSHA256] = true
		reasons[result.ReasonCode] = true
	}
	metrics.UniqueReasonCount = len(reasons)
	return metrics, nil
}

func deriveAutonomousRepairGovernanceCoverage(results []AutonomousRepairGovernanceResult) AutonomousRepairGovernanceCoverage {
	byID := make(map[string]AutonomousRepairGovernanceResult, len(results))
	for _, result := range results {
		byID[result.CaseID] = result
	}
	authorized := func(id string, writes int) bool {
		result, ok := byID[id]
		return ok && result.Authorized && result.TerminalState == "authorized" && result.WriteCount == writes && result.Conformant
	}
	denied := func(id string, writes int) bool {
		result, ok := byID[id]
		return ok && !result.Authorized && result.TerminalState == "denied" && result.WriteCount == writes && result.Conformant
	}
	all := func(checks ...bool) bool {
		for _, check := range checks {
			if !check {
				return false
			}
		}
		return true
	}
	return AutonomousRepairGovernanceCoverage{
		SoleControlOptIn: all(
			denied("sole-auto-merge-default-off", 0),
			authorized("sole-auto-merge-opted-in", 1),
		),
		TeamHumanCodeownerExactHead: authorized("team-independent-exact-head", 1),
		AutomatedReviewerDenied: all(
			denied("team-automated-reviewer-word", 0),
			denied("team-bot-reviewer", 0),
			denied("team-ao-prefix-reviewer", 0),
			denied("team-codex-prefix-reviewer", 0),
		),
		ExternalUnknownDraftOnly: all(
			authorized("external-create-draft", 1),
			denied("external-merge-denied", 0),
			authorized("unknown-draft-only", 1),
			denied("unknown-merge-denied", 0),
		),
		ClassSensitiveReadyPR: all(
			authorized("team-open-ready-pr", 1),
			authorized("sole-open-ready-pr", 1),
			denied("ready-transition-denied", 0),
		),
		ExactForkBranchPrerequisite: all(
			denied("draft-fork-prerequisite-absent", 0),
			denied("draft-fork-prerequisite-mismatch", 0),
			denied("draft-branch-prerequisite-absent", 0),
			denied("draft-branch-prerequisite-mismatch", 0),
		),
		DuplicateDraftIdempotence: authorized("external-reuse-exact-draft", 0),
		ConflictReread: all(
			authorized("draft-conflict-exact-reread", 1),
			denied("draft-conflict-missing-reread", 1),
			denied("draft-conflict-identity-drift", 1),
			denied("draft-conflict-ambiguous-reread", 1),
		),
		PostWriteReadback: all(
			denied("draft-post-create-readback-drift", 1),
			denied("fork-post-create-readback-drift", 1),
			denied("branch-post-push-readback-drift", 1),
		),
		PerWriteExpiry: all(
			denied("expiry-before-fork-write", 0),
			denied("expiry-before-branch-write", 1),
			denied("expiry-before-draft-write", 0),
		),
		PermanentWriteDenial: all(
			denied("force-update-denied", 0),
			denied("upstream-push-denied", 0),
			denied("issue-mutation-denied", 0),
			denied("review-denied", 0),
			denied("merge-denied", 0),
		),
		ProtectedChecksDigest: all(
			denied("protected-path-denied", 0),
			denied("failed-check-denied", 0),
			denied("action-digest-mismatch", 0),
		),
	}
}

func CanonicalAutonomousRepairGovernanceEvaluationDigest(report AutonomousRepairGovernanceEvaluation) (string, error) {
	report.CanonicalDigest = ""
	return canonicalGovernanceDigest(report)
}

func validateAutonomousRepairGovernanceEvaluation(report AutonomousRepairGovernanceEvaluation) error {
	if report.SchemaVersion != autonomousRepairGovernanceEvaluationSchema ||
		report.EvaluationID != autonomousRepairGovernanceEvaluationID ||
		report.Status != "passed" {
		return fmt.Errorf("invalid governance evaluation identity or status")
	}
	expectedCrucible := AutonomousRepairGovernanceCrucibleProvenance{
		Repository:           "uesugitorachiyo/ao-crucible",
		Commit:               "130043b8b9c8c71eb1b4ae4856faec4d240a38dd",
		SuitePath:            "examples/autonomous-repair/valid/stage5-governance-assurance.json",
		SuiteSHA256:          "d5a750c80bacc4f2ddda11c3993c786290a21f4348d7e1f44f7fb71535b49511",
		SchemaPath:           "docs/contracts/crucible-autonomous-repair-governance-assurance-v1.schema.json",
		SchemaSHA256:         "cbd392db1704e1a1e5e6f17cf5d403192bf2a8f57d74ae09aa76d36c3270898f",
		ImplementationPath:   "internal/crucible/autonomous_repair_governance_assurance.go",
		ImplementationSHA256: "6663867a63852316148fd8dc6f582c57680c30baece2556a3cdd11d3341bdc2b",
		SuiteCanonicalDigest: "fe6aa5a08f6a670609feea0639929dd28937adba94804c29b85ca8471d406872",
	}
	if report.Provenance.Crucible != expectedCrucible {
		return fmt.Errorf("governance evaluation Crucible provenance mismatch")
	}
	if report.Subjects != (AutonomousRepairGovernanceSubjects{
		CovenantCommit: "561c167c57199913d4e2fa2692c21da68a2ecae6",
		AO2Commit:      "627c53f952bae5a638ce25ed934a81d01527a9f1",
	}) {
		return fmt.Errorf("governance evaluation subject mismatch")
	}
	if report.SafetyBoundary != (AutonomousRepairGovernanceSafety{Mode: "fixture_only"}) {
		return fmt.Errorf("governance evaluation exceeded fixture-only safety boundary")
	}
	if report.PromotionRequested {
		return fmt.Errorf("governance evaluation must not request promotion")
	}
	if report.ResultsSHA256 != autonomousRepairGovernanceResultsSHA256 {
		return fmt.Errorf("governance evaluation results digest mismatch")
	}
	resultsDigest, err := canonicalGovernanceDigest(report.Results)
	if err != nil {
		return err
	}
	if resultsDigest != report.ResultsSHA256 {
		return fmt.Errorf("governance evaluation results do not replay")
	}
	metrics, err := EvaluateAutonomousRepairGovernanceResults(report.Results)
	if err != nil {
		return err
	}
	if metrics != expectedAutonomousRepairGovernanceMetrics || report.Metrics != metrics {
		return fmt.Errorf("governance evaluation metrics mismatch")
	}
	derivedCoverage := deriveAutonomousRepairGovernanceCoverage(report.Results)
	if derivedCoverage != expectedAutonomousRepairGovernanceCoverage || report.Coverage != derivedCoverage {
		return fmt.Errorf("governance evaluation coverage mismatch")
	}
	digest, err := CanonicalAutonomousRepairGovernanceEvaluationDigest(report)
	if err != nil {
		return err
	}
	if !lowercaseSHA256.MatchString(report.CanonicalDigest) || digest != report.CanonicalDigest {
		return fmt.Errorf("governance evaluation canonical digest mismatch")
	}
	return nil
}

func requireAutonomousRepairGovernanceFields(data []byte) error {
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	required := map[string][]string{
		"":                    {"schema_version", "evaluation_id", "status", "provenance", "subjects", "safety_boundary", "results", "results_sha256", "metrics", "coverage", "promotion_requested", "canonical_digest"},
		"provenance":          {"crucible"},
		"provenance.crucible": {"repository", "commit", "suite_path", "suite_sha256", "schema_path", "schema_sha256", "implementation_path", "implementation_sha256", "suite_canonical_digest"},
		"subjects":            {"covenant_commit", "ao2_commit"},
		"safety_boundary":     {"mode", "live_provider_used", "network_used", "credentials_used", "github_mutation", "sibling_repo_mutation", "provider_call_performed"},
		"metrics":             {"case_count", "authorized_count", "denied_count", "total_write_count", "max_write_count", "unique_reason_count", "authorized_zero_write_count", "denied_after_write_count", "conformance_failures", "duplicate_case_ids", "duplicate_input_digests"},
		"coverage":            {"sole_control_opt_in", "team_human_codeowner_exact_head", "automated_reviewer_denied", "external_unknown_draft_only", "class_sensitive_ready_pr", "exact_fork_branch_prerequisite", "duplicate_draft_idempotence", "conflict_reread", "post_write_readback", "per_write_expiry", "permanent_write_denial", "protected_checks_digest"},
	}
	for path, fields := range required {
		object, err := governanceObjectAt(document, path)
		if err != nil {
			return err
		}
		for _, field := range fields {
			if _, ok := object[field]; !ok {
				return fmt.Errorf("governance evaluation %s missing required field %q", path, field)
			}
		}
	}
	results, ok := document["results"].([]any)
	if !ok {
		return fmt.Errorf("governance evaluation results must be an array")
	}
	for index, value := range results {
		result, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("governance evaluation result %d must be an object", index)
		}
		for _, field := range []string{"case_id", "input_sha256", "authorized", "reason_code", "write_count", "terminal_state", "conformant"} {
			if _, ok := result[field]; !ok {
				return fmt.Errorf("governance evaluation result %d missing required field %q", index, field)
			}
		}
	}
	return nil
}

func governanceObjectAt(root map[string]any, path string) (map[string]any, error) {
	current := root
	if path == "" {
		return current, nil
	}
	start := 0
	for index := 0; index <= len(path); index++ {
		if index != len(path) && path[index] != '.' {
			continue
		}
		field := path[start:index]
		next, ok := current[field].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("governance evaluation %q must be an object", path[:index])
		}
		current = next
		start = index + 1
	}
	return current, nil
}

func canonicalGovernanceDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	var canonical any
	if err := json.Unmarshal(data, &canonical); err != nil {
		return "", err
	}
	data, err = json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
