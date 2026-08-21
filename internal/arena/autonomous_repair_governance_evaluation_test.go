package arena

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const autonomousRepairGovernanceEvaluationFixture = "examples/reports/valid/autonomous-repair-governance-evaluation.json"

func TestAutonomousRepairGovernanceEvaluationRecomputesPassingMetrics(t *testing.T) {
	report := loadAutonomousRepairGovernanceEvaluation(t)
	derived, err := EvaluateAutonomousRepairGovernanceResults(report.Results)
	if err != nil {
		t.Fatal(err)
	}
	if derived != report.Metrics {
		t.Fatalf("derived metrics = %#v, fixture metrics %#v", derived, report.Metrics)
	}
	if report.Status != "passed" ||
		report.Metrics.CaseCount != 47 ||
		report.Metrics.AuthorizedCount != 10 ||
		report.Metrics.DeniedCount != 37 ||
		report.Metrics.TotalWriteCount != 16 ||
		report.Metrics.MaxWriteCount != 2 ||
		report.Metrics.UniqueReasonCount != 32 ||
		report.Metrics.AuthorizedZeroWriteCount != 2 ||
		report.Metrics.DeniedAfterWriteCount != 7 ||
		report.Metrics.ConformanceFailures != 0 ||
		report.Metrics.DuplicateCaseIDs != 0 ||
		report.Metrics.DuplicateInputDigests != 0 {
		t.Fatalf("unexpected governance metrics: %#v", report.Metrics)
	}
}

func TestAutonomousRepairGovernanceEvaluationPinsMergedProducerAndSubjects(t *testing.T) {
	report := loadAutonomousRepairGovernanceEvaluation(t)
	if report.Provenance.Crucible.Commit != "130043b8b9c8c71eb1b4ae4856faec4d240a38dd" ||
		report.Subjects.CovenantCommit != "561c167c57199913d4e2fa2692c21da68a2ecae6" ||
		report.Subjects.AO2Commit != "627c53f952bae5a638ce25ed934a81d01527a9f1" {
		t.Fatalf("merged provenance = %#v subjects = %#v", report.Provenance, report.Subjects)
	}
	if report.Provenance.Crucible.SuiteSHA256 != "d5a750c80bacc4f2ddda11c3993c786290a21f4348d7e1f44f7fb71535b49511" ||
		report.Provenance.Crucible.SchemaSHA256 != "cbd392db1704e1a1e5e6f17cf5d403192bf2a8f57d74ae09aa76d36c3270898f" ||
		report.Provenance.Crucible.ImplementationSHA256 != "6663867a63852316148fd8dc6f582c57680c30baece2556a3cdd11d3341bdc2b" ||
		report.Provenance.Crucible.SuiteCanonicalDigest != "fe6aa5a08f6a670609feea0639929dd28937adba94804c29b85ca8471d406872" {
		t.Fatalf("producer digest provenance = %#v", report.Provenance.Crucible)
	}
}

func TestAutonomousRepairGovernanceEvaluationCoversWaveCBoundaries(t *testing.T) {
	report := loadAutonomousRepairGovernanceEvaluation(t)
	want := AutonomousRepairGovernanceCoverage{
		SoleControlOptIn:            true,
		TeamHumanCodeownerExactHead: true,
		AutomatedReviewerDenied:     true,
		ExternalUnknownDraftOnly:    true,
		ClassSensitiveReadyPR:       true,
		ExactForkBranchPrerequisite: true,
		DuplicateDraftIdempotence:   true,
		ConflictReread:              true,
		PostWriteReadback:           true,
		PerWriteExpiry:              true,
		PermanentWriteDenial:        true,
		ProtectedChecksDigest:       true,
	}
	if derived := deriveAutonomousRepairGovernanceCoverage(report.Results); derived != want {
		t.Fatalf("derived coverage = %#v", derived)
	}
	if report.Coverage != want {
		t.Fatalf("coverage = %#v", report.Coverage)
	}

	altered := append([]AutonomousRepairGovernanceResult(nil), report.Results...)
	for index := range altered {
		if altered[index].CaseID == "team-bot-reviewer" {
			altered[index].CaseID = "unrelated-denied-case"
			break
		}
	}
	if deriveAutonomousRepairGovernanceCoverage(altered).AutomatedReviewerDenied {
		t.Fatal("derived automated-reviewer coverage survived a missing required case")
	}
}

func TestAutonomousRepairGovernanceEvaluationRejectsRedigestedResultSubstitution(t *testing.T) {
	document := readAutonomousRepairGovernanceEvaluationDocument(t)
	results := document["results"].([]any)
	first := results[0].(map[string]any)
	first["authorized"] = true
	first["reason_code"] = "sole_control_auto_merge"
	first["write_count"] = float64(1)
	first["terminal_state"] = "authorized"
	setGovernanceEvaluationResultsDigest(t, document)
	setGovernanceEvaluationDigest(t, document)
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAndValidateAutonomousRepairGovernanceEvaluation(data); err == nil {
		t.Fatal("accepted redigested result substitution")
	}
}

func TestAutonomousRepairGovernanceEvaluationRejectsMetricAndSafetyDrift(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"metrics", func(d map[string]any) {
			d["metrics"].(map[string]any)["conformance_failures"] = float64(1)
		}},
		{"coverage", func(d map[string]any) {
			d["coverage"].(map[string]any)["per_write_expiry"] = false
		}},
		{"safety", func(d map[string]any) {
			d["safety_boundary"].(map[string]any)["github_mutation"] = true
		}},
		{"promotion", func(d map[string]any) {
			d["promotion_requested"] = true
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			document := readAutonomousRepairGovernanceEvaluationDocument(t)
			tt.mutate(document)
			setGovernanceEvaluationDigest(t, document)
			data, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeAndValidateAutonomousRepairGovernanceEvaluation(data); err == nil {
				t.Fatal("accepted governance evaluation drift")
			}
		})
	}
}

func TestAutonomousRepairGovernanceEvaluationRejectsMissingUnknownAndDuplicateFields(t *testing.T) {
	document := readAutonomousRepairGovernanceEvaluationDocument(t)
	delete(document["safety_boundary"].(map[string]any), "provider_call_performed")
	setGovernanceEvaluationDigest(t, document)
	missing, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAndValidateAutonomousRepairGovernanceEvaluation(missing); err == nil {
		t.Fatal("accepted missing false-valued safety field")
	}

	document = readAutonomousRepairGovernanceEvaluationDocument(t)
	document["unexpected"] = true
	setGovernanceEvaluationDigest(t, document)
	unknown, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAndValidateAutonomousRepairGovernanceEvaluation(unknown); err == nil {
		t.Fatal("accepted unknown field")
	}

	duplicate := []byte(`{"schema_version":"ao.arena.autonomous-repair-governance-evaluation.v1","schema_version":"duplicate"}`)
	if _, err := DecodeAndValidateAutonomousRepairGovernanceEvaluation(duplicate); err == nil {
		t.Fatal("accepted duplicate field")
	}
}

func TestAutonomousRepairGovernanceEvaluationCanonicalDigestReplays(t *testing.T) {
	first := loadAutonomousRepairGovernanceEvaluation(t)
	second := loadAutonomousRepairGovernanceEvaluation(t)
	firstDigest, err := CanonicalAutonomousRepairGovernanceEvaluationDigest(first)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := CanonicalAutonomousRepairGovernanceEvaluationDigest(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != first.CanonicalDigest || firstDigest != secondDigest {
		t.Fatalf("digest replay = %q, %q; fixture = %q", firstDigest, secondDigest, first.CanonicalDigest)
	}
}

func TestAutonomousRepairGovernanceEvaluationRejectsMalformedOversizedAndSymlinkFiles(t *testing.T) {
	root := t.TempDir()
	malformedPath := filepath.Join(root, "malformed.json")
	if err := os.WriteFile(malformedPath, []byte(`{"status":`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAndValidateAutonomousRepairGovernanceEvaluation(malformedPath); err == nil {
		t.Fatal("accepted malformed evaluation")
	}

	oversizedPath := filepath.Join(root, "oversized.json")
	if err := os.WriteFile(oversizedPath, []byte(strings.Repeat(" ", autonomousRepairGovernanceEvaluationLimit+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAndValidateAutonomousRepairGovernanceEvaluation(oversizedPath); err == nil {
		t.Fatal("accepted oversized evaluation")
	}

	fixture := filepath.Join(repoRoot(t), autonomousRepairGovernanceEvaluationFixture)
	symlinkPath := filepath.Join(root, "evaluation.json")
	requireTestSymlink(t, fixture, symlinkPath)
	if _, err := LoadAndValidateAutonomousRepairGovernanceEvaluation(symlinkPath); err == nil {
		t.Fatal("accepted symlinked evaluation")
	}
}

func TestAutonomousRepairGovernanceEvaluationPublicSchemaIsStrict(t *testing.T) {
	root := repoRoot(t)
	schemaBytes, err := os.ReadFile(filepath.Join(
		root,
		"docs/contracts/arena-autonomous-repair-governance-evaluation-v1.schema.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	var schemaDocument testJSONSchema
	if err := json.Unmarshal(schemaBytes, &schemaDocument); err != nil {
		t.Fatal(err)
	}
	valid := readAutonomousRepairGovernanceEvaluationDocument(t)
	if err := validateTestSchemaValue(valid, &schemaDocument, &schemaDocument, "$"); err != nil {
		t.Fatalf("valid fixture failed public schema: %v", err)
	}
	for _, mutate := range []func(map[string]any){
		func(d map[string]any) { d["unknown"] = true },
		func(d map[string]any) { delete(d["results"].([]any)[0].(map[string]any), "input_sha256") },
		func(d map[string]any) { d["results"].([]any)[0].(map[string]any)["authorized"] = true },
		func(d map[string]any) { d["metrics"].(map[string]any)["case_count"] = float64(48) },
		func(d map[string]any) { d["status"] = "failed" },
		func(d map[string]any) { d["canonical_digest"] = strings.Repeat("0", 64) },
	} {
		document := readAutonomousRepairGovernanceEvaluationDocument(t)
		mutate(document)
		if err := validateTestSchemaValue(document, &schemaDocument, &schemaDocument, "$"); err == nil {
			t.Fatal("public schema accepted invalid governance evaluation")
		}
	}
}

func loadAutonomousRepairGovernanceEvaluation(t *testing.T) AutonomousRepairGovernanceEvaluation {
	t.Helper()
	report, err := LoadAndValidateAutonomousRepairGovernanceEvaluation(filepath.Join(
		repoRoot(t),
		autonomousRepairGovernanceEvaluationFixture,
	))
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func readAutonomousRepairGovernanceEvaluationDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), autonomousRepairGovernanceEvaluationFixture))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func setGovernanceEvaluationResultsDigest(t *testing.T, document map[string]any) {
	t.Helper()
	data, err := json.Marshal(document["results"])
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	document["results_sha256"] = hex.EncodeToString(sum[:])
}

func setGovernanceEvaluationDigest(t *testing.T, document map[string]any) {
	t.Helper()
	delete(document, "canonical_digest")
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	document["canonical_digest"] = hex.EncodeToString(sum[:])
}
