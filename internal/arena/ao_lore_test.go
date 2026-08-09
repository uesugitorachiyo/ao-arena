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

func TestCompareAOLoreMatchedAttempts(t *testing.T) {
	dir := t.TempDir()
	baseline := aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000)
	challenger := aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000)
	manifest := aoLoreManifest("pdf", writeAOLoreAttempt(t, dir, baseline), writeAOLoreAttempt(t, dir, challenger))
	input := writeAOLoreManifest(t, dir, manifest)
	out := filepath.Join(dir, "comparison.json")

	report, err := CompareAOLore(input, out)
	if err != nil {
		t.Fatal(err)
	}
	if report.Winner != "challenger" || report.Result != "win" || len(report.Pairs) != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
	pair := report.Pairs[0]
	if pair.Baseline.AttemptID != "baseline-pdf" || pair.Challenger.RoleConfiguration.Navigator != "navigator-local" {
		t.Fatalf("attempt evidence was not retained: %#v", pair)
	}
	if pair.FixtureCorpusSHA256 != strings.Repeat("a", 64) || pair.SourceInputSHA256 != strings.Repeat("d", 64) {
		t.Fatalf("matched digest bindings were not retained: %#v", pair)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatal(err)
	}
}

func TestCompareAOLorePreservesNonApplicableMetricsAndTies(t *testing.T) {
	dir := t.TempDir()
	baseline := aoLoreAttempt("baseline-text", "baseline", "text", 8500)
	challenger := aoLoreAttempt("challenger-text", "challenger", "text", 8500)
	baseline.Metrics.ParserCostEfficiency = AOLoreMetric{Applicable: false, ScoreBasisPoints: nil, Reason: "not measured for direct text"}
	challenger.Metrics.ParserCostEfficiency = baseline.Metrics.ParserCostEfficiency
	manifest := aoLoreManifest("text", writeAOLoreAttempt(t, dir, baseline), writeAOLoreAttempt(t, dir, challenger))

	report, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Winner != "tie" || report.Result != "tie" {
		t.Fatalf("expected visible tie, got %#v", report)
	}
	metric := report.Pairs[0].Baseline.Metrics.ParserCostEfficiency
	if metric.Applicable || metric.ScoreBasisPoints != nil {
		t.Fatalf("non-applicable metric was zeroed: %#v", metric)
	}
}

func TestCompareAOLoreRejectsDigestAndMatchedPairDrift(t *testing.T) {
	t.Run("attempt digest", func(t *testing.T) {
		dir := t.TempDir()
		baseline := writeAOLoreAttempt(t, dir, aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000))
		baseline.SHA256 = strings.Repeat("0", 64)
		manifest := aoLoreManifest("pdf", baseline, writeAOLoreAttempt(t, dir, aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000)))
		if _, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json")); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
			t.Fatalf("expected digest mismatch, got %v", err)
		}
	})

	t.Run("corpus", func(t *testing.T) {
		dir := t.TempDir()
		baseline := aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000)
		challenger := aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000)
		challenger.FixtureCorpusSHA256 = strings.Repeat("b", 64)
		manifest := aoLoreManifest("pdf", writeAOLoreAttempt(t, dir, baseline), writeAOLoreAttempt(t, dir, challenger))
		if _, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json")); err == nil || !strings.Contains(err.Error(), "matched") {
			t.Fatalf("expected matched-pair error, got %v", err)
		}
	})

	t.Run("applicability", func(t *testing.T) {
		dir := t.TempDir()
		baseline := aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000)
		challenger := aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000)
		challenger.Metrics.ParserCostEfficiency = AOLoreMetric{Applicable: false, Reason: "excluded"}
		manifest := aoLoreManifest("pdf", writeAOLoreAttempt(t, dir, baseline), writeAOLoreAttempt(t, dir, challenger))
		if _, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json")); err == nil || !strings.Contains(err.Error(), "applicability") {
			t.Fatalf("expected applicability error, got %v", err)
		}
	})
}

func TestCompareAOLoreRetainsFailuresExclusionsAndIneligibility(t *testing.T) {
	dir := t.TempDir()
	baseline := aoLoreAttempt("baseline-image", "baseline", "image", 8000)
	baseline.Eligible = false
	baseline.IneligibilityReasons = []string{"trace contract unavailable"}
	challenger := aoLoreAttempt("challenger-image", "challenger", "image", 9000)
	challenger.Failures = []string{"ocr timeout"}
	challenger.Exclusions = []string{"handwriting subset excluded by fixed corpus contract"}
	manifest := aoLoreManifest("image", writeAOLoreAttempt(t, dir, baseline), writeAOLoreAttempt(t, dir, challenger))

	report, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatal(err)
	}
	if report.Winner != "tie" || report.Pairs[0].Baseline.ComparisonScore != 0 || report.Pairs[0].Challenger.ComparisonScore != 0 {
		t.Fatalf("failed/ineligible attempts must score zero: %#v", report)
	}
	if len(report.Pairs[0].Baseline.IneligibilityReasons) != 1 || len(report.Pairs[0].Challenger.Failures) != 1 || len(report.Pairs[0].Challenger.Exclusions) != 1 {
		t.Fatalf("failure visibility lost: %#v", report.Pairs[0])
	}
}

func TestCompareAOLoreRejectsUnsafeReferenceAndExistingOutput(t *testing.T) {
	dir := t.TempDir()
	baseline := writeAOLoreAttempt(t, dir, aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000))
	challenger := writeAOLoreAttempt(t, dir, aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000))
	manifest := aoLoreManifest("pdf", baseline, challenger)
	manifest.Pairs[0].Baseline.Path = "../escape.json"
	if _, err := CompareAOLore(writeAOLoreManifest(t, dir, manifest), filepath.Join(dir, "out.json")); err == nil {
		t.Fatal("expected unsafe path rejection")
	}

	dir = t.TempDir()
	manifest = aoLoreManifest("pdf", writeAOLoreAttempt(t, dir, aoLoreAttempt("baseline-pdf", "baseline", "pdf", 8000)), writeAOLoreAttempt(t, dir, aoLoreAttempt("challenger-pdf", "challenger", "pdf", 9000)))
	input := writeAOLoreManifest(t, dir, manifest)
	out := filepath.Join(dir, "out.json")
	if err := os.WriteFile(out, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CompareAOLore(input, out); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite rejection, got %v", err)
	}
}

func aoLoreAttempt(attemptID, systemID, format string, score int) AOLoreAttempt {
	value := score
	return AOLoreAttempt{
		SchemaVersion:         "ao.arena.ao-lore-attempt.v0.1",
		AttemptID:             attemptID,
		SystemID:              systemID,
		Format:                format,
		FixtureCorpusSHA256:   strings.Repeat("a", 64),
		WorkloadProfileSHA256: strings.Repeat("c", 64),
		SourceInputSHA256:     strings.Repeat("d", 64),
		ResultSHA256:          strings.Repeat("e", 64),
		Metrics: AOLoreMetrics{
			ParserFidelity:           AOLoreMetric{Applicable: true, ScoreBasisPoints: &value, Reason: "fixed parser benchmark"},
			ParserCostEfficiency:     AOLoreMetric{Applicable: true, ScoreBasisPoints: &value, Reason: "fixed cost benchmark"},
			CoverageEfficiency:       AOLoreMetric{Applicable: true, ScoreBasisPoints: &value, Reason: "fixed coverage benchmark"},
			RoleConfigurationQuality: AOLoreMetric{Applicable: true, ScoreBasisPoints: &value, Reason: "fixed role matrix"},
		},
		RoleConfiguration:    AOLoreRoleConfiguration{Parser: "parser-local", Distiller: "distiller-local", Navigator: "navigator-local", Synthesizer: "synthesizer-local"},
		Eligible:             true,
		IneligibilityReasons: []string{},
		Failures:             []string{},
		Exclusions:           []string{},
		TraceIntegrity:       true,
	}
}

func aoLoreManifest(format string, baseline, challenger AOLoreEvidenceReference) AOLoreEvaluationManifest {
	return AOLoreEvaluationManifest{
		SchemaVersion:            "ao.arena.ao-lore-evaluation.v0.1",
		EvaluationID:             "ao-lore-test-evaluation",
		MetricWeightsBasisPoints: AOLoreMetricWeights{ParserFidelity: 4000, ParserCostEfficiency: 2000, CoverageEfficiency: 3000, RoleConfigurationQuality: 1000},
		Pairs:                    []AOLorePairReference{{Format: format, Baseline: baseline, Challenger: challenger}},
	}
}

func writeAOLoreAttempt(t *testing.T, dir string, attempt AOLoreAttempt) AOLoreEvidenceReference {
	t.Helper()
	body, err := json.MarshalIndent(attempt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	name := attempt.AttemptID + ".json"
	if err := os.WriteFile(filepath.Join(dir, name), body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	return AOLoreEvidenceReference{Path: name, SHA256: hex.EncodeToString(digest[:])}
}

func writeAOLoreManifest(t *testing.T, dir string, manifest AOLoreEvaluationManifest) string {
	t.Helper()
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
