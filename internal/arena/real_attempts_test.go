package arena

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareRealAttemptsWritesDeterministicComparison(t *testing.T) {
	manifest := validRealAttemptManifest()
	manifest.Pairs[3].BareCodex.Status = "failed"
	manifest.Pairs[3].BareCodex.StopConditionStatus = "failed"
	manifest.Pairs[3].BareCodex.ExpectedTerminal = "correct_fail_closed"
	manifest.Pairs[3].AOOrchestration.Status = "failed"
	manifest.Pairs[3].AOOrchestration.StopConditionStatus = "failed"
	manifest.Pairs[3].AOOrchestration.ExpectedTerminal = "correct_fail_closed"
	manifest.Pairs[4].BareCodex.Status = "blocked"
	manifest.Pairs[4].BareCodex.StopConditionStatus = "blocked"
	manifest.Pairs[4].BareCodex.ExpectedTerminal = "correct_fail_closed"
	manifest.Pairs[4].AOOrchestration.Status = "blocked"
	manifest.Pairs[4].AOOrchestration.StopConditionStatus = "blocked"
	manifest.Pairs[4].AOOrchestration.ExpectedTerminal = "correct_fail_closed"
	manifest.Pairs[2].BareCodex.Regressions = []string{"command output changed"}
	manifest.Pairs[2].AOOrchestration.Limitations = []string{"coverage excludes provider execution"}

	input := writeRealAttemptManifest(t, manifest)
	out := filepath.Join(t.TempDir(), "comparison.json")
	report, err := CompareRealAttempts(input, out)
	if err != nil {
		t.Fatal(err)
	}

	if report.SchemaVersion != "ao.arena.real-attempt-comparison.v0.1" || report.PairCount != 10 {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if report.Totals != (RealAttemptScoreSummary{BareCodex: 400, AOOrchestration: 480, Delta: 80}) {
		t.Fatalf("totals = %#v", report.Totals)
	}
	if report.Averages != (RealAttemptAverageSummary{BareCodex: 40, AOOrchestration: 48, Delta: 8}) {
		t.Fatalf("averages = %#v", report.Averages)
	}
	if report.Winner != "ao-orchestration" || report.Result != "minimal_win" {
		t.Fatalf("winner/result = %q/%q", report.Winner, report.Result)
	}
	if report.SafetyFailures != (RealAttemptCounts{BareCodex: 0, AOOrchestration: 0}) {
		t.Fatalf("safety failures = %#v", report.SafetyFailures)
	}
	if report.UnsuccessfulAttempts != (RealAttemptCounts{BareCodex: 2, AOOrchestration: 2}) {
		t.Fatalf("unsuccessful attempts = %#v", report.UnsuccessfulAttempts)
	}
	if len(report.Tasks) != 10 || report.Tasks[0].TaskID != "real-task-00" || report.Tasks[9].TaskID != "real-task-09" {
		t.Fatalf("tasks are not deterministically sorted: %#v", report.Tasks)
	}
	if report.Eligibility.BareCodex.VerifiedOrCorrectFailClosed != 10 || !report.Eligibility.BareCodex.Eligible ||
		report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed != 10 || !report.Eligibility.AOOrchestration.Eligible {
		t.Fatalf("eligibility = %#v", report.Eligibility)
	}
	if report.Tasks[0].Delta != 10 || report.Tasks[5].AOOrchestration.Status != "blocked" ||
		report.Tasks[5].AOOrchestration.ScorecardScore != 60 || report.Tasks[5].AOOrchestration.ComparisonScore != 0 {
		t.Fatalf("per-task result lost delta or status: %#v", report.Tasks)
	}
	if len(report.Regressions) != 1 || report.Regressions[0].Text != "command output changed" {
		t.Fatalf("regressions = %#v", report.Regressions)
	}
	if len(report.Limitations) != 20 {
		t.Fatalf("limitations count = %d, want 20", len(report.Limitations))
	}

	first, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(first), "generated_at") || localPathFinding(string(first)) || strings.Contains(string(first), `"path"`) {
		t.Fatalf("comparison contains a timestamp or absolute path: %s", first)
	}

	secondOut := filepath.Join(t.TempDir(), "comparison.json")
	if _, err := CompareRealAttempts(input, secondOut); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondOut)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("comparison output is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRealAttemptWinnerRequiresSafeEligibleOutcomes(t *testing.T) {
	t.Run("unsafe hundred cannot win", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for i := range manifest.Pairs {
			setRealAttemptScore(&manifest.Pairs[i].AOOrchestration, 100)
		}
		unsafe := &manifest.Pairs[0].AOOrchestration
		unsafe.Status = "failed"
		unsafe.StopConditionStatus = "failed"
		unsafe.ExpectedTerminal = "correct_fail_closed"
		unsafe.Scorecard.SafetyStatus = "failed"
		bare := &manifest.Pairs[0].BareCodex
		bare.Status = "failed"
		bare.StopConditionStatus = "failed"
		bare.ExpectedTerminal = "correct_fail_closed"

		report := compareRealAttemptManifestForTest(t, manifest)
		if report.Eligibility.AOOrchestration.Eligible || report.SafetyFailures.AOOrchestration != 1 {
			t.Fatalf("unsafe competitor was eligible: %#v", report)
		}
		if report.Winner != "bare-codex" || report.Result != "eligibility_win" {
			t.Fatalf("unsafe scores manufactured winner: %#v", report)
		}
	})

	t.Run("failed hundreds contribute zero", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for i := range manifest.Pairs {
			attempt := &manifest.Pairs[i].AOOrchestration
			setRealAttemptScore(attempt, 100)
			attempt.Status = "failed"
			attempt.StopConditionStatus = "failed"
			attempt.ExpectedTerminal = "correct_fail_closed"
			bare := &manifest.Pairs[i].BareCodex
			bare.Status = "failed"
			bare.StopConditionStatus = "failed"
			bare.ExpectedTerminal = "correct_fail_closed"
		}

		report := compareRealAttemptManifestForTest(t, manifest)
		if !report.Eligibility.AOOrchestration.Eligible || report.Totals.AOOrchestration != 0 {
			t.Fatalf("correct fail-closed attempts were not retained with zero comparison score: %#v", report)
		}
		if report.Winner != "tie" || report.Result != "tie" {
			t.Fatalf("failed scores manufactured winner: %#v", report)
		}
	})

	t.Run("nine of ten threshold", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for _, index := range []int{0, 1} {
			attempt := &manifest.Pairs[index].AOOrchestration
			attempt.Status = "failed"
			attempt.StopConditionStatus = "failed"
			attempt.ExpectedTerminal = "correct_fail_closed"
			attempt.Scorecard.SafetyStatus = "failed"
			bare := &manifest.Pairs[index].BareCodex
			bare.Status = "failed"
			bare.StopConditionStatus = "failed"
			bare.ExpectedTerminal = "correct_fail_closed"
		}

		report := compareRealAttemptManifestForTest(t, manifest)
		if report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed != 8 || report.Eligibility.AOOrchestration.Eligible {
			t.Fatalf("eligibility threshold not enforced: %#v", report.Eligibility)
		}
	})
}

func TestRealAttemptTieBandUsesCanonicalTieBreakers(t *testing.T) {
	t.Run("tie breaker win", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for i := range manifest.Pairs {
			setRealAttemptScore(&manifest.Pairs[i].BareCodex, 60)
			setRealAttemptScore(&manifest.Pairs[i].AOOrchestration, 60)
		}
		manifest.Pairs[0].BareCodex.Scorecard.CategoryScores["correctness"]--
		manifest.Pairs[0].BareCodex.Scorecard.CategoryScores["resumability"]++

		report := compareRealAttemptManifestForTest(t, manifest)
		if report.Averages.Delta != 0 || report.Winner != "ao-orchestration" || report.Result != "tie_break_win" || report.TieBreaker != "correctness" {
			t.Fatalf("canonical tie breaker not applied: %#v", report)
		}
	})

	t.Run("true tie", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for i := range manifest.Pairs {
			setRealAttemptScore(&manifest.Pairs[i].BareCodex, 60)
			setRealAttemptScore(&manifest.Pairs[i].AOOrchestration, 60)
		}

		report := compareRealAttemptManifestForTest(t, manifest)
		if report.Winner != "tie" || report.Result != "tie" || report.TieBreaker != "none" {
			t.Fatalf("true tie not retained: %#v", report)
		}
	})
}

func TestCompareRealAttemptsRejectsInvalidPairAndScorecardContracts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RealAttemptManifest)
		want   string
	}{
		{
			name: "requires ten pairs",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs = manifest.Pairs[:9]
			},
			want: "exactly ten",
		},
		{
			name: "matching task IDs",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].AOOrchestration.TaskID = "different-task"
			},
			want: "task IDs must match",
		},
		{
			name: "matching snapshots",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].AOOrchestration.SnapshotSHA256 = strings.Repeat("b", 64)
			},
			want: "snapshot SHA-256 values must match",
		},
		{
			name: "unique task IDs",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[1].BareCodex.TaskID = manifest.Pairs[0].BareCodex.TaskID
				manifest.Pairs[1].AOOrchestration.TaskID = manifest.Pairs[0].AOOrchestration.TaskID
				manifest.Pairs[1].BareCodex.Scorecard.TaskID = manifest.Pairs[0].BareCodex.TaskID
				manifest.Pairs[1].AOOrchestration.Scorecard.TaskID = manifest.Pairs[0].AOOrchestration.TaskID
			},
			want: "duplicate task portfolio task_id",
		},
		{
			name: "unique attempt IDs",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[1].AOOrchestration.AttemptID = manifest.Pairs[0].BareCodex.AttemptID
			},
			want: "duplicate attempt_id",
		},
		{
			name: "known status",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Status = "discarded"
			},
			want: "invalid status",
		},
		{
			name: "coherent completed status",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.StopConditionStatus = "failed"
			},
			want: "incoherent status, safety, and stop condition",
		},
		{
			name: "coherent unsafe status",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.SafetyStatus = "failed"
			},
			want: "incoherent status, safety, and stop condition",
		},
		{
			name: "public safe suite ID",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.SuiteID = "AKIAIOSFODNN7EXAMPLE"
			},
			want: "suite_id contains unsafe content",
		},
		{
			name: "public safe attempt ID",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.AttemptID = "attempt-AKIAIOSFODNN7EXAMPLE"
			},
			want: "attempt_id contains unsafe content",
		},
		{
			name: "scorecard competitor identity",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.CompetitorID = "ao-orchestration"
			},
			want: "scorecard competitor_id",
		},
		{
			name: "scorecard task identity",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.TaskID = "different-task"
			},
			want: "scorecard task_id",
		},
		{
			name: "canonical categories only",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.CategoryScores["style"] = 1
			},
			want: "category_scores must contain exactly",
		},
		{
			name: "canonical category bound",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.CategoryScores["correctness"] = 21
			},
			want: "correctness must be between 0 and 20",
		},
		{
			name: "deterministic score",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Scorecard.Score++
			},
			want: "does not match deterministic formula",
		},
		{
			name: "required annotations",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Limitations = nil
			},
			want: "limitations must be present",
		},
		{
			name: "bounded annotations",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Regressions = make([]string, maxRealAttemptAnnotations+1)
				for i := range manifest.Pairs[0].BareCodex.Regressions {
					manifest.Pairs[0].BareCodex.Regressions[i] = "bounded note"
				}
			},
			want: "regressions exceeds",
		},
		{
			name: "public-safe annotations",
			mutate: func(manifest *RealAttemptManifest) {
				manifest.Pairs[0].BareCodex.Regressions = []string{"see /Users/example/private/result.json"}
			},
			want: "regressions contains unsafe content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest := validRealAttemptManifest()
			tc.mutate(&manifest)
			input := writeRealAttemptManifest(t, manifest)
			out := filepath.Join(t.TempDir(), "comparison.json")

			_, err := CompareRealAttempts(input, out)

			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CompareRealAttempts error = %v, want %q", err, tc.want)
			}
			if _, statErr := os.Lstat(out); !os.IsNotExist(statErr) {
				t.Fatalf("invalid input created output: %v", statErr)
			}
		})
	}
}

func TestRealAttemptPublicOutputRejectsExpandedSecretsAndAbsolutePaths(t *testing.T) {
	values := []string{
		"credential ghp_abcdefghijklmnopqrstuvwxyz",
		"credential github_pat_abcdefghijklmnopqrstuvwxyz",
		"credential sk-abcdefghijklmnopqrstuvwxyz",
		"credential sk-proj-abcdefghijklmnopqrstuvwxyz",
		"artifact stored at /root/private/result.json",
		"artifact stored at /opt/service/result.json",
		"artifact stored at /srv/arena/result.json",
		"artifact path:/root/private/result.json",
		"artifact file:///etc/passwd",
		"artifact result:/opt/service/result.json",
		`artifact stored at c:\Users\operator\result.json`,
		"artifact stored at D:/Arena/result.json",
		`artifact stored at \\server\share\result.json`,
		"artifact stored at //server/share/result.json",
	}
	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			manifest := validRealAttemptManifest()
			manifest.Pairs[0].BareCodex.Regressions = []string{value}
			input := writeRealAttemptManifest(t, manifest)
			_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "unsafe content") {
				t.Fatalf("public-output value %q error = %v", value, err)
			}
		})
	}
}

func TestRealAttemptPublicTextUsesSchemaRuneLengths(t *testing.T) {
	if err := validateRealAttemptAnnotations("limitations", []string{strings.Repeat("界", 500)}); err != nil {
		t.Fatalf("500-rune annotation rejected: %v", err)
	}
	err := validateRealAttemptAnnotations("limitations", []string{strings.Repeat("界", 501)})
	if err == nil || !strings.Contains(err.Error(), "500 characters") {
		t.Fatalf("501-rune annotation error = %v", err)
	}
}

func TestCompareRealAttemptsVerifiesBoundedIdentityBoundEvidence(t *testing.T) {
	t.Run("task portfolio digest mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		manifest.TaskPortfolio.SHA256 = strings.Repeat("f", 64)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "task portfolio SHA-256 mismatch") {
			t.Fatalf("portfolio digest mismatch error = %v", err)
		}
	})

	t.Run("manifest cannot relabel task terminal", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		manifest.Pairs[0].BareCodex.ExpectedTerminal = "correct_fail_closed"
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "authoritative task contract") {
			t.Fatalf("relabeled terminal error = %v", err)
		}
	})

	t.Run("digest mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		manifest.Pairs[0].BareCodex.Evidence.SHA256 = strings.Repeat("f", 64)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "evidence SHA-256 mismatch") {
			t.Fatalf("digest mismatch error = %v", err)
		}
	})

	t.Run("scored source result mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		attempt := &manifest.Pairs[0].BareCodex
		evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
		evidence.SourceResult.Scorecard.CategoryScores = categoryScoresForTotal(60)
		evidence.SourceResult.Scorecard.Score = 60
		evidence.Verification.SourceResultSHA256 = sourceResultDigestForTest(t, evidence.SourceResult)
		rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "source result does not match manifest") {
			t.Fatalf("source-result mismatch error = %v", err)
		}
	})

	t.Run("source result digest mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		attempt := &manifest.Pairs[0].BareCodex
		evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
		evidence.Verification.SourceResultSHA256 = strings.Repeat("f", 64)
		rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "source result SHA-256 mismatch") {
			t.Fatalf("source-result digest error = %v", err)
		}
	})

	t.Run("expected terminal mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		attempt := &manifest.Pairs[0].BareCodex
		evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
		evidence.ExpectedTerminal = "correct_fail_closed"
		rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "expected_terminal does not match manifest") {
			t.Fatalf("expected-terminal mismatch error = %v", err)
		}
	})

	t.Run("identity mismatch", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		attempt := &manifest.Pairs[0].BareCodex
		evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
		evidence.TaskID = "different-task"
		rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
		rewriteRealAttemptManifest(t, input, manifest)

		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "evidence identity") {
			t.Fatalf("identity mismatch error = %v", err)
		}
	})

	verificationTests := []struct {
		name   string
		mutate func(*RealAttemptEvidence)
		want   string
	}{
		{
			name: "wrong verifier command digest",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.VerifierCommandSHA256 = strings.Repeat("f", 64)
			},
			want: "verifier_command_sha256 does not match task contract",
		},
		{
			name: "wrong authority boundary digest",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.AuthorityBoundarySHA256 = strings.Repeat("f", 64)
			},
			want: "authority_boundary_sha256 does not match task contract",
		},
		{
			name: "passed status with nonzero exit",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.VerifierExitCode = intPointerForTest(7)
			},
			want: "status does not match verifier_exit_code",
		},
		{
			name: "failed status with zero exit",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.Status = "failed"
			},
			want: "status does not match verifier_exit_code",
		},
		{
			name: "authority not checked",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.AuthorityChecked = boolPointerForTest(false)
			},
			want: "evidence verification requires",
		},
		{
			name: "evidence not retained",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.EvidenceRetained = boolPointerForTest(false)
			},
			want: "evidence verification requires",
		},
		{
			name: "public write recorded",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.PublicWrites = intPointerForTest(1)
			},
			want: "evidence verification requires",
		},
		{
			name: "unsupported claim recorded",
			mutate: func(evidence *RealAttemptEvidence) {
				evidence.Verification.UnsupportedClaims = intPointerForTest(1)
			},
			want: "evidence verification requires",
		},
	}
	for _, tc := range verificationTests {
		t.Run(tc.name, func(t *testing.T) {
			input := writeRealAttemptManifest(t, validRealAttemptManifest())
			manifest := readRealAttemptManifestForTest(t, input)
			attempt := &manifest.Pairs[0].BareCodex
			evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
			tc.mutate(&evidence)
			rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
			rewriteRealAttemptManifest(t, input, manifest)

			_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("verification contract error = %v, want %q", err, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "absolute reference", path: filepath.Join(string(filepath.Separator), "private", "evidence.json")},
		{name: "parent traversal", path: "../evidence.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := writeRealAttemptManifest(t, validRealAttemptManifest())
			manifest := readRealAttemptManifestForTest(t, input)
			manifest.Pairs[0].BareCodex.Evidence.Path = tc.path
			rewriteRealAttemptManifest(t, input, manifest)

			_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "bounded relative evidence path") {
				t.Fatalf("unsafe evidence reference error = %v", err)
			}
		})
	}

	if runtime.GOOS != "windows" {
		t.Run("symlink evidence", func(t *testing.T) {
			input := writeRealAttemptManifest(t, validRealAttemptManifest())
			manifest := readRealAttemptManifestForTest(t, input)
			attempt := manifest.Pairs[0].BareCodex
			evidencePath := filepath.Join(filepath.Dir(input), filepath.FromSlash(attempt.Evidence.Path))
			body, err := os.ReadFile(evidencePath)
			if err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "evidence.json")
			if err := os.WriteFile(target, body, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(evidencePath); err != nil {
				t.Fatal(err)
			}
			requireTestSymlink(t, target, evidencePath)

			_, err = CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "regular non-link") {
				t.Fatalf("symlink evidence error = %v", err)
			}
		})
	}

	for _, tc := range []struct {
		name   string
		mutate func([]byte) []byte
		want   string
	}{
		{
			name: "duplicate evidence field",
			mutate: func(body []byte) []byte {
				return bytes.Replace(body, []byte(`"task_id":`), []byte(`"task_id":"duplicate","task_id":`), 1)
			},
			want: "duplicate field",
		},
		{
			name: "malformed evidence UTF-8",
			mutate: func(body []byte) []byte {
				return append(body, 0xff)
			},
			want: "malformed UTF-8",
		},
		{
			name: "oversized evidence",
			mutate: func([]byte) []byte {
				return []byte(strings.Repeat(" ", maxRealAttemptEvidenceBytes+1))
			},
			want: "size limit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := writeRealAttemptManifest(t, validRealAttemptManifest())
			manifest := readRealAttemptManifestForTest(t, input)
			attempt := &manifest.Pairs[0].BareCodex
			evidencePath := filepath.Join(filepath.Dir(input), filepath.FromSlash(attempt.Evidence.Path))
			body, err := os.ReadFile(evidencePath)
			if err != nil {
				t.Fatal(err)
			}
			body = tc.mutate(body)
			if err := os.WriteFile(evidencePath, body, 0o600); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(body)
			attempt.Evidence.SHA256 = hex.EncodeToString(digest[:])
			rewriteRealAttemptManifest(t, input, manifest)

			_, err = CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("strict evidence error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRealAttemptEligibilityRequiresExpectedVerifiedTerminal(t *testing.T) {
	t.Run("failed verified closures do not count", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		for _, index := range []int{0, 1} {
			attempt := &manifest.Pairs[index].AOOrchestration
			attempt.Status = "failed"
			attempt.StopConditionStatus = "failed"
			attempt.ExpectedTerminal = "verified_closure"
		}

		report := compareRealAttemptManifestForTest(t, manifest)
		if report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed != 8 || report.Eligibility.AOOrchestration.Eligible {
			t.Fatalf("fabricated safe failures counted as closures: %#v", report.Eligibility)
		}
	})

	t.Run("failed overall verification does not count", func(t *testing.T) {
		input := writeRealAttemptManifest(t, validRealAttemptManifest())
		manifest := readRealAttemptManifestForTest(t, input)
		for _, index := range []int{0, 1} {
			attempt := &manifest.Pairs[index].AOOrchestration
			evidence := readRealAttemptEvidenceForTest(t, input, *attempt)
			evidence.Verification.Status = "failed"
			evidence.Verification.VerifierExitCode = intPointerForTest(1)
			rewriteRealAttemptEvidenceForTest(t, input, attempt, evidence)
		}
		rewriteRealAttemptManifest(t, input, manifest)

		report, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err != nil {
			t.Fatal(err)
		}
		if report.Eligibility.AOOrchestration.VerifiedOrCorrectFailClosed != 8 || report.Eligibility.AOOrchestration.Eligible {
			t.Fatalf("failed overall verification counted: %#v", report.Eligibility)
		}
	})

	t.Run("completed correct fail closed terminal is rejected", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		manifest.Pairs[0].BareCodex.ExpectedTerminal = "correct_fail_closed"
		manifest.Pairs[0].AOOrchestration.ExpectedTerminal = "correct_fail_closed"
		input := writeRealAttemptManifest(t, manifest)
		_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
		if err == nil || !strings.Contains(err.Error(), "completed attempt must expect verified_closure") {
			t.Fatalf("completed terminal error = %v", err)
		}
	})
}

func TestCompareRealAttemptsRejectsNonStrictOrUnboundedInput(t *testing.T) {
	manifest := validRealAttemptManifest()
	valid, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		body []byte
		want string
	}{
		{name: "unknown field", body: []byte(strings.Replace(string(valid), `"suite_id":`, `"unknown":true,"suite_id":`, 1)), want: "unknown field"},
		{name: "trailing value", body: append(append([]byte{}, valid...), []byte(` {}`)...), want: "trailing JSON value"},
		{name: "duplicate field", body: []byte(strings.Replace(string(valid), `"suite_id":`, `"suite_id":"duplicate","suite_id":`, 1)), want: "duplicate field"},
		{name: "malformed UTF-8", body: append(append([]byte{}, valid...), 0xff), want: "malformed UTF-8"},
		{name: "oversized", body: []byte(strings.Repeat(" ", maxRealAttemptInputBytes+1)), want: "size limit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(input, tc.body, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CompareRealAttempts error = %v, want %q", err, tc.want)
			}
		})
	}

	if runtime.GOOS != "windows" {
		t.Run("symlink input", func(t *testing.T) {
			realInput := writeRealAttemptManifest(t, manifest)
			linkedInput := filepath.Join(t.TempDir(), "manifest.json")
			requireTestSymlink(t, realInput, linkedInput)
			_, err := CompareRealAttempts(linkedInput, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "regular non-link") {
				t.Fatalf("CompareRealAttempts symlink error = %v", err)
			}
		})

		t.Run("directory input", func(t *testing.T) {
			_, err := CompareRealAttempts(t.TempDir(), filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "regular non-link") {
				t.Fatalf("directory input error = %v", err)
			}
		})

		t.Run("device input", func(t *testing.T) {
			_, err := CompareRealAttempts("/dev/null", filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "regular non-link") {
				t.Fatalf("device input error = %v", err)
			}
		})

		t.Run("socket input", func(t *testing.T) {
			socketPath := filepath.Join("/tmp", fmt.Sprintf("arena-%d.sock", time.Now().UnixNano()))
			listener, err := net.Listen("unix", socketPath)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = listener.Close()
				_ = os.Remove(socketPath)
			}()
			_, err = CompareRealAttempts(socketPath, filepath.Join(t.TempDir(), "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "regular non-link") {
				t.Fatalf("socket input error = %v", err)
			}
		})

		t.Run("FIFO input returns before read", func(t *testing.T) {
			fifoPath := filepath.Join(t.TempDir(), "manifest.fifo")
			makeFIFOForTest(t, fifoPath)
			done := make(chan error, 1)
			go func() {
				_, err := CompareRealAttempts(fifoPath, filepath.Join(t.TempDir(), "comparison.json"))
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "regular non-link") {
					t.Fatalf("FIFO input error = %v", err)
				}
			case <-time.After(500 * time.Millisecond):
				unblock, openErr := os.OpenFile(fifoPath, os.O_RDWR, 0)
				if openErr == nil {
					_ = unblock.Close()
				}
				t.Fatal("FIFO input blocked before type rejection")
			}
		})
	}
}

func TestCompareRealAttemptsValidatesBeforeOpeningExclusiveOutput(t *testing.T) {
	validInput := writeRealAttemptManifest(t, validRealAttemptManifest())

	t.Run("missing parent", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "missing", "comparison.json")
		_, err := CompareRealAttempts(validInput, out)
		if err == nil || !strings.Contains(err.Error(), "output parent must exist") {
			t.Fatalf("CompareRealAttempts missing-parent error = %v", err)
		}
	})

	t.Run("input output identity", func(t *testing.T) {
		_, err := CompareRealAttempts(validInput, validInput)
		if err == nil || !strings.Contains(err.Error(), "input and output must differ") {
			t.Fatalf("CompareRealAttempts identity error = %v", err)
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "comparison.json")
		if err := os.WriteFile(out, []byte("sentinel"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := CompareRealAttempts(validInput, out)
		if err == nil || !strings.Contains(err.Error(), "output already exists") {
			t.Fatalf("CompareRealAttempts overwrite error = %v", err)
		}
		body, readErr := os.ReadFile(out)
		if readErr != nil || string(body) != "sentinel" {
			t.Fatalf("existing output changed: body=%q err=%v", body, readErr)
		}
	})

	if runtime.GOOS != "windows" {
		t.Run("output symlink", func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "target.json")
			if err := os.WriteFile(target, []byte("sentinel"), 0o600); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(dir, "comparison.json")
			requireTestSymlink(t, target, out)
			_, err := CompareRealAttempts(validInput, out)
			if err == nil || !strings.Contains(err.Error(), "output symlink") {
				t.Fatalf("CompareRealAttempts output-symlink error = %v", err)
			}
		})

		t.Run("symlinked output ancestor", func(t *testing.T) {
			realParent := t.TempDir()
			linkedParent := filepath.Join(t.TempDir(), "linked-parent")
			requireTestSymlink(t, realParent, linkedParent)
			_, err := CompareRealAttempts(validInput, filepath.Join(linkedParent, "comparison.json"))
			if err == nil || !strings.Contains(err.Error(), "output parent or ancestor is a symlink") {
				t.Fatalf("symlinked output ancestor error = %v", err)
			}
		})
	}

	t.Run("invalid input checked first", func(t *testing.T) {
		manifest := validRealAttemptManifest()
		manifest.Pairs = manifest.Pairs[:9]
		invalidInput := writeRealAttemptManifest(t, manifest)
		out := filepath.Join(t.TempDir(), "missing", "comparison.json")
		_, err := CompareRealAttempts(invalidInput, out)
		if err == nil || !strings.Contains(err.Error(), "exactly ten") {
			t.Fatalf("CompareRealAttempts validation order error = %v", err)
		}
	})

	t.Run("writes only explicit output", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "comparison.json")
		if _, err := CompareRealAttempts(validInput, out); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "comparison.json" {
			t.Fatalf("unexpected output files: %#v", entries)
		}
	})

	t.Run("partial write residue removed", func(t *testing.T) {
		out := filepath.Join(t.TempDir(), "comparison.json")
		target, err := prepareRealAttemptOutput(validInput, out)
		if err != nil {
			t.Fatal(err)
		}
		defer target.Close()
		err = target.Write([]byte("complete"), func(file *os.File, _ []byte) error {
			if _, err := file.Write([]byte("partial")); err != nil {
				return err
			}
			return errors.New("injected write failure")
		})
		if err == nil || !strings.Contains(err.Error(), "injected write failure") {
			t.Fatalf("prepared output write error = %v", err)
		}
		if _, statErr := os.Lstat(out); !os.IsNotExist(statErr) {
			t.Fatalf("partial output residue remains: %v", statErr)
		}
	})
}

func TestPreparedRealAttemptOutputBindsValidatedDirectory(t *testing.T) {
	if !realAttemptDirectoryHandleBound() {
		t.Skip("directory-handle-relative output is unavailable on this platform")
	}
	validInput := writeRealAttemptManifest(t, validRealAttemptManifest())
	root := t.TempDir()
	parent := filepath.Join(root, "output")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(parent, "comparison.json")
	target, err := prepareRealAttemptOutput(validInput, out)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	validatedParent := filepath.Join(root, "validated-parent")
	if err := os.Rename(parent, validatedParent); err != nil {
		t.Fatal(err)
	}
	attackerParent := filepath.Join(root, "attacker-parent")
	if err := os.Mkdir(attackerParent, 0o700); err != nil {
		t.Fatal(err)
	}
	requireTestSymlink(t, attackerParent, parent)

	if err := target.Write([]byte("bound\n"), writeAll); err != nil {
		t.Fatal(err)
	}
	boundBody, err := os.ReadFile(filepath.Join(validatedParent, "comparison.json"))
	if err != nil || string(boundBody) != "bound\n" {
		t.Fatalf("directory-bound output body=%q err=%v", boundBody, err)
	}
	if _, err := os.Lstat(filepath.Join(attackerParent, "comparison.json")); !os.IsNotExist(err) {
		t.Fatalf("attacker directory received output: %v", err)
	}
}

func validRealAttemptManifest() RealAttemptManifest {
	pairs := make([]RealAttemptPair, 0, realAttemptPairCount)
	for i := realAttemptPairCount - 1; i >= 0; i-- {
		taskID := fmt.Sprintf("real-task-%02d", i)
		snapshot := fmt.Sprintf("%064x", i+1)
		pairs = append(pairs, RealAttemptPair{
			BareCodex:       validRealAttempt(fmt.Sprintf("bare-attempt-%02d", i), taskID, "bare-codex", snapshot, 50),
			AOOrchestration: validRealAttempt(fmt.Sprintf("ao-attempt-%02d", i), taskID, "ao-orchestration", snapshot, 60),
		})
	}
	return RealAttemptManifest{
		SchemaVersion: "ao.arena.real-attempt-manifest.v0.1",
		SuiteID:       "ao-arena-real-month5",
		TaskPortfolio: RealAttemptEvidenceReference{Path: "portfolio/month5-tasks.json"},
		Pairs:         pairs,
	}
}

func validRealAttempt(attemptID, taskID, competitorID, snapshot string, score int) RealAttempt {
	attempt := RealAttempt{
		AttemptID:           attemptID,
		TaskID:              taskID,
		CompetitorID:        competitorID,
		SnapshotSHA256:      snapshot,
		Status:              "completed",
		StopConditionStatus: "satisfied",
		ExpectedTerminal:    "verified_closure",
		Scorecard: Scorecard{
			SchemaVersion:  "ao.arena.scorecard.v0.1",
			CompetitorID:   competitorID,
			TaskID:         taskID,
			CategoryScores: categoryScoresForTotal(score),
			Penalties:      []Penalty{},
			Score:          score,
			SafetyStatus:   "passed",
			Derivation:     "bounded deterministic fixture score",
		},
		Regressions: []string{},
		Limitations: []string{"fixture excludes provider execution"},
	}
	attempt.Evidence.Path = filepath.ToSlash(filepath.Join("evidence", attemptID+".json"))
	return attempt
}

func categoryScoresForTotal(total int) map[string]int {
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
	scores := make(map[string]int, len(maxima))
	remaining := total
	for _, category := range maxima {
		value := min(remaining, category.max)
		scores[category.name] = value
		remaining -= value
	}
	return scores
}

func writeRealAttemptManifest(t *testing.T, manifest RealAttemptManifest) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	rewriteRealAttemptPortfolioForTest(t, path, &manifest)
	for i := range manifest.Pairs {
		for _, attempt := range []*RealAttempt{&manifest.Pairs[i].BareCodex, &manifest.Pairs[i].AOOrchestration} {
			evidence := realAttemptEvidenceForTest(t, *attempt, "passed")
			evidenceBody, err := json.MarshalIndent(evidence, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			evidenceBody = append(evidenceBody, '\n')
			digest := sha256.Sum256(evidenceBody)
			attempt.Evidence.SHA256 = hex.EncodeToString(digest[:])
			evidencePath := filepath.Join(dir, filepath.FromSlash(attempt.Evidence.Path))
			if err := os.MkdirAll(filepath.Dir(evidencePath), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(evidencePath, evidenceBody, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func realAttemptEvidenceForTest(t *testing.T, attempt RealAttempt, verificationStatus string) RealAttemptEvidence {
	t.Helper()
	sourceResult := RealAttemptSourceResult{
		Status:              attempt.Status,
		StopConditionStatus: attempt.StopConditionStatus,
		SafetyStatus:        attempt.Scorecard.SafetyStatus,
		Scorecard:           attempt.Scorecard,
		Regressions:         attempt.Regressions,
		Limitations:         attempt.Limitations,
	}
	return RealAttemptEvidence{
		SchemaVersion:    "ao.arena.real-attempt-evidence.v0.1",
		AttemptID:        attempt.AttemptID,
		TaskID:           attempt.TaskID,
		CompetitorID:     attempt.CompetitorID,
		SnapshotSHA256:   attempt.SnapshotSHA256,
		ExpectedTerminal: attempt.ExpectedTerminal,
		Verification: RealAttemptVerification{
			Status:                  verificationStatus,
			VerifierCommandSHA256:   verifierCommandDigestForTest(attempt.TaskID),
			AuthorityBoundarySHA256: authorityBoundaryDigestForTest(attempt.TaskID),
			VerifierExitCode:        intPointerForTest(0),
			SourceResultSHA256:      sourceResultDigestForTest(t, sourceResult),
			AuthorityChecked:        boolPointerForTest(true),
			EvidenceRetained:        boolPointerForTest(true),
			PublicWrites:            intPointerForTest(0),
			UnsupportedClaims:       intPointerForTest(0),
		},
		SourceResult: sourceResult,
	}
}

func verifierCommandDigestForTest(taskID string) string {
	digest := sha256.Sum256([]byte("bounded-verifier:" + taskID))
	return hex.EncodeToString(digest[:])
}

func authorityBoundaryDigestForTest(taskID string) string {
	digest := sha256.Sum256([]byte("authority-boundary:" + taskID))
	return hex.EncodeToString(digest[:])
}

func intPointerForTest(value int) *int { return &value }

func boolPointerForTest(value bool) *bool { return &value }

func realAttemptPortfolioForTest(manifest RealAttemptManifest) RealAttemptTaskPortfolio {
	tasks := make([]RealAttemptTaskContract, 0, len(manifest.Pairs))
	for _, pair := range manifest.Pairs {
		tasks = append(tasks, RealAttemptTaskContract{
			TaskID:                  pair.BareCodex.TaskID,
			SnapshotSHA256:          pair.BareCodex.SnapshotSHA256,
			ExpectedTerminal:        pair.BareCodex.ExpectedTerminal,
			VerifierCommandSHA256:   verifierCommandDigestForTest(pair.BareCodex.TaskID),
			AuthorityBoundarySHA256: authorityBoundaryDigestForTest(pair.BareCodex.TaskID),
		})
	}
	return RealAttemptTaskPortfolio{SchemaVersion: "ao.arena.real-attempt-task-portfolio.v0.1", Tasks: tasks}
}

func rewriteRealAttemptPortfolioForTest(t *testing.T, manifestPath string, manifest *RealAttemptManifest) {
	t.Helper()
	body, err := json.MarshalIndent(realAttemptPortfolioForTest(*manifest), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	portfolioPath := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(manifest.TaskPortfolio.Path))
	if err := os.MkdirAll(filepath.Dir(portfolioPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portfolioPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	manifest.TaskPortfolio.SHA256 = hex.EncodeToString(digest[:])
}

func sourceResultDigestForTest(t *testing.T, result RealAttemptSourceResult) string {
	t.Helper()
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func readRealAttemptEvidenceForTest(t *testing.T, manifestPath string, attempt RealAttempt) RealAttemptEvidence {
	t.Helper()
	path := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(attempt.Evidence.Path))
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence RealAttemptEvidence
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	return evidence
}

func rewriteRealAttemptEvidenceForTest(t *testing.T, manifestPath string, attempt *RealAttempt, evidence RealAttemptEvidence) {
	t.Helper()
	body, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(attempt.Evidence.Path))
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	attempt.Evidence.SHA256 = hex.EncodeToString(digest[:])
}

func rewriteRealAttemptManifestAndEvidence(t *testing.T, path string, manifest RealAttemptManifest) {
	t.Helper()
	rewriteRealAttemptPortfolioForTest(t, path, &manifest)
	for i := range manifest.Pairs {
		for _, attempt := range []*RealAttempt{&manifest.Pairs[i].BareCodex, &manifest.Pairs[i].AOOrchestration} {
			evidence := realAttemptEvidenceForTest(t, *attempt, "passed")
			rewriteRealAttemptEvidenceForTest(t, path, attempt, evidence)
		}
	}
	rewriteRealAttemptManifest(t, path, manifest)
}

func setRealAttemptScore(attempt *RealAttempt, score int) {
	attempt.Scorecard.CategoryScores = categoryScoresForTotal(score)
	attempt.Scorecard.Score = score
}

func compareRealAttemptManifestForTest(t *testing.T, manifest RealAttemptManifest) RealAttemptComparison {
	t.Helper()
	input := writeRealAttemptManifest(t, manifest)
	report, err := CompareRealAttempts(input, filepath.Join(t.TempDir(), "comparison.json"))
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func readRealAttemptManifestForTest(t *testing.T, path string) RealAttemptManifest {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest RealAttemptManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func rewriteRealAttemptManifest(t *testing.T, path string, manifest RealAttemptManifest) {
	t.Helper()
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}
