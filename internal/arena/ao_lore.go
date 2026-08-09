package arena

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	maxAOLoreManifestBytes = 256 * 1024
	maxAOLoreAttemptBytes  = 128 * 1024
	maxAOLorePairs         = 50
)

var aoLoreFormats = map[string]bool{
	"pdf": true, "markdown": true, "html": true, "docx": true, "text": true, "image": true,
}

type AOLoreEvidenceReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type AOLoreEvaluationManifest struct {
	SchemaVersion            string                `json:"schema_version"`
	EvaluationID             string                `json:"evaluation_id"`
	MetricWeightsBasisPoints AOLoreMetricWeights   `json:"metric_weights_basis_points"`
	Pairs                    []AOLorePairReference `json:"pairs"`
}

type AOLorePairReference struct {
	Format     string                  `json:"format"`
	Baseline   AOLoreEvidenceReference `json:"baseline"`
	Challenger AOLoreEvidenceReference `json:"challenger"`
}

type AOLoreMetricWeights struct {
	ParserFidelity           int `json:"parser_fidelity"`
	ParserCostEfficiency     int `json:"parser_cost_efficiency"`
	CoverageEfficiency       int `json:"coverage_efficiency"`
	RoleConfigurationQuality int `json:"role_configuration_quality"`
}

type AOLoreAttempt struct {
	SchemaVersion         string                  `json:"schema_version"`
	AttemptID             string                  `json:"attempt_id"`
	SystemID              string                  `json:"system_id"`
	Format                string                  `json:"format"`
	FixtureCorpusSHA256   string                  `json:"fixture_corpus_sha256"`
	WorkloadProfileSHA256 string                  `json:"workload_profile_sha256"`
	SourceInputSHA256     string                  `json:"source_input_sha256"`
	ResultSHA256          string                  `json:"result_sha256"`
	Metrics               AOLoreMetrics           `json:"metrics"`
	RoleConfiguration     AOLoreRoleConfiguration `json:"role_configuration"`
	Eligible              bool                    `json:"eligible"`
	IneligibilityReasons  []string                `json:"ineligibility_reasons"`
	Failures              []string                `json:"failures"`
	Exclusions            []string                `json:"exclusions"`
	TraceIntegrity        bool                    `json:"trace_integrity"`
}

type AOLoreMetrics struct {
	ParserFidelity           AOLoreMetric `json:"parser_fidelity"`
	ParserCostEfficiency     AOLoreMetric `json:"parser_cost_efficiency"`
	CoverageEfficiency       AOLoreMetric `json:"coverage_efficiency"`
	RoleConfigurationQuality AOLoreMetric `json:"role_configuration_quality"`
}

type AOLoreMetric struct {
	Applicable       bool   `json:"applicable"`
	ScoreBasisPoints *int   `json:"score_basis_points"`
	Reason           string `json:"reason"`
}

type AOLoreRoleConfiguration struct {
	Parser      string `json:"parser"`
	Distiller   string `json:"distiller"`
	Navigator   string `json:"navigator"`
	Synthesizer string `json:"synthesizer"`
}

type AOLoreComparison struct {
	SchemaVersion            string                 `json:"schema_version"`
	EvaluationID             string                 `json:"evaluation_id"`
	MetricWeightsBasisPoints AOLoreMetricWeights    `json:"metric_weights_basis_points"`
	Pairs                    []AOLorePairComparison `json:"pairs"`
	BaselineSystemID         string                 `json:"baseline_system_id"`
	ChallengerSystemID       string                 `json:"challenger_system_id"`
	BaselineTotal            int                    `json:"baseline_total"`
	ChallengerTotal          int                    `json:"challenger_total"`
	Winner                   string                 `json:"winner"`
	Result                   string                 `json:"result"`
	Authority                string                 `json:"authority"`
}

type AOLorePairComparison struct {
	Format                string               `json:"format"`
	FixtureCorpusSHA256   string               `json:"fixture_corpus_sha256"`
	WorkloadProfileSHA256 string               `json:"workload_profile_sha256"`
	SourceInputSHA256     string               `json:"source_input_sha256"`
	Baseline              AOLoreAttemptOutcome `json:"baseline"`
	Challenger            AOLoreAttemptOutcome `json:"challenger"`
	Winner                string               `json:"winner"`
	Result                string               `json:"result"`
}

type AOLoreAttemptOutcome struct {
	AttemptID            string                  `json:"attempt_id"`
	SystemID             string                  `json:"system_id"`
	ResultSHA256         string                  `json:"result_sha256"`
	Metrics              AOLoreMetrics           `json:"metrics"`
	RoleConfiguration    AOLoreRoleConfiguration `json:"role_configuration"`
	Eligible             bool                    `json:"eligible"`
	IneligibilityReasons []string                `json:"ineligibility_reasons"`
	Failures             []string                `json:"failures"`
	Exclusions           []string                `json:"exclusions"`
	TraceIntegrity       bool                    `json:"trace_integrity"`
	RawCompositeScore    int                     `json:"raw_composite_score"`
	ComparisonScore      int                     `json:"comparison_score"`
}

func CompareAOLore(inputPath, outputPath string) (AOLoreComparison, error) {
	manifest, attempts, err := loadAOLoreEvaluation(inputPath)
	if err != nil {
		return AOLoreComparison{}, err
	}
	report, err := buildAOLoreComparison(manifest, attempts)
	if err != nil {
		return AOLoreComparison{}, err
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return AOLoreComparison{}, err
	}
	body = append(body, '\n')
	target, err := prepareRealAttemptOutput(inputPath, outputPath)
	if err != nil {
		return AOLoreComparison{}, err
	}
	defer target.Close()
	if err := target.Write(body, writeAll); err != nil {
		return AOLoreComparison{}, err
	}
	return report, nil
}

func loadAOLoreEvaluation(inputPath string) (AOLoreEvaluationManifest, map[string]AOLoreAttempt, error) {
	var manifest AOLoreEvaluationManifest
	root, manifestName, err := openRealAttemptInputRoot(inputPath)
	if err != nil {
		return manifest, nil, err
	}
	defer root.Close()
	if _, err := readStrictBoundedJSONFromRoot(root, manifestName, "AO Lore evaluation manifest", maxAOLoreManifestBytes, &manifest); err != nil {
		return manifest, nil, err
	}
	if manifest.SchemaVersion != "ao.arena.ao-lore-evaluation.v0.1" {
		return manifest, nil, fmt.Errorf("invalid AO Lore evaluation schema_version")
	}
	if err := validatePublicIdentifier("evaluation_id", manifest.EvaluationID); err != nil {
		return manifest, nil, err
	}
	if err := validateAOLoreWeights(manifest.MetricWeightsBasisPoints); err != nil {
		return manifest, nil, err
	}
	if len(manifest.Pairs) == 0 || len(manifest.Pairs) > maxAOLorePairs {
		return manifest, nil, fmt.Errorf("AO Lore evaluation must contain between 1 and %d pairs", maxAOLorePairs)
	}
	attempts := make(map[string]AOLoreAttempt, len(manifest.Pairs)*2)
	attemptIDs := map[string]bool{}
	paths := map[string]bool{}
	for i, pair := range manifest.Pairs {
		if !aoLoreFormats[pair.Format] {
			return manifest, nil, fmt.Errorf("pair %d has unsupported format", i)
		}
		for label, reference := range map[string]AOLoreEvidenceReference{"baseline": pair.Baseline, "challenger": pair.Challenger} {
			if err := validateEvidenceReference(RealAttemptEvidenceReference(reference)); err != nil {
				return manifest, nil, fmt.Errorf("pair %d %s: %w", i, label, err)
			}
			if paths[reference.Path] {
				return manifest, nil, fmt.Errorf("attempt evidence path must be unique")
			}
			paths[reference.Path] = true
			attempt, err := loadAOLoreAttempt(root, reference)
			if err != nil {
				return manifest, nil, fmt.Errorf("pair %d %s: %w", i, label, err)
			}
			if attempt.Format != pair.Format {
				return manifest, nil, fmt.Errorf("pair %d %s format does not match manifest", i, label)
			}
			if attemptIDs[attempt.AttemptID] {
				return manifest, nil, fmt.Errorf("duplicate AO Lore attempt_id %q", attempt.AttemptID)
			}
			attemptIDs[attempt.AttemptID] = true
			attempts[reference.Path] = attempt
		}
	}
	return manifest, attempts, nil
}

func loadAOLoreAttempt(root *os.Root, reference AOLoreEvidenceReference) (AOLoreAttempt, error) {
	var attempt AOLoreAttempt
	body, err := readStrictBoundedJSONFromRoot(root, reference.Path, "AO Lore attempt", maxAOLoreAttemptBytes, &attempt)
	if err != nil {
		return attempt, err
	}
	digest := sha256.Sum256(body)
	if hex.EncodeToString(digest[:]) != reference.SHA256 {
		return attempt, fmt.Errorf("attempt SHA-256 mismatch")
	}
	if err := validateAOLoreAttempt(attempt); err != nil {
		return attempt, err
	}
	return attempt, nil
}

func validateAOLoreWeights(weights AOLoreMetricWeights) error {
	values := []int{weights.ParserFidelity, weights.ParserCostEfficiency, weights.CoverageEfficiency, weights.RoleConfigurationQuality}
	total := 0
	for _, value := range values {
		if value < 0 || value > 10000 {
			return fmt.Errorf("AO Lore metric weight must be between 0 and 10000 basis points")
		}
		total += value
	}
	if total != 10000 {
		return fmt.Errorf("AO Lore metric weights must total 10000 basis points")
	}
	return nil
}

type aoLoreMetricEntry struct {
	name   string
	metric AOLoreMetric
	weight int
}

func aoLoreMetricEntries(metrics AOLoreMetrics, weights AOLoreMetricWeights) []aoLoreMetricEntry {
	return []aoLoreMetricEntry{
		{name: "parser_fidelity", metric: metrics.ParserFidelity, weight: weights.ParserFidelity},
		{name: "parser_cost_efficiency", metric: metrics.ParserCostEfficiency, weight: weights.ParserCostEfficiency},
		{name: "coverage_efficiency", metric: metrics.CoverageEfficiency, weight: weights.CoverageEfficiency},
		{name: "role_configuration_quality", metric: metrics.RoleConfigurationQuality, weight: weights.RoleConfigurationQuality},
	}
}

func validateAOLoreAttempt(attempt AOLoreAttempt) error {
	if attempt.SchemaVersion != "ao.arena.ao-lore-attempt.v0.1" {
		return fmt.Errorf("invalid AO Lore attempt schema_version")
	}
	if err := validatePublicIdentifier("attempt_id", attempt.AttemptID); err != nil {
		return err
	}
	if err := validatePublicIdentifier("system_id", attempt.SystemID); err != nil {
		return err
	}
	if !aoLoreFormats[attempt.Format] {
		return fmt.Errorf("AO Lore attempt has unsupported format")
	}
	for label, value := range map[string]string{
		"fixture_corpus_sha256":   attempt.FixtureCorpusSHA256,
		"workload_profile_sha256": attempt.WorkloadProfileSHA256,
		"source_input_sha256":     attempt.SourceInputSHA256,
		"result_sha256":           attempt.ResultSHA256,
	} {
		if !sha256Pattern.MatchString(value) {
			return fmt.Errorf("%s must be 64 lowercase hexadecimal characters", label)
		}
	}
	for _, entry := range aoLoreMetricEntries(attempt.Metrics, AOLoreMetricWeights{}) {
		if err := validatePublicText(entry.name+" reason", entry.metric.Reason); err != nil {
			return err
		}
		if entry.metric.Applicable {
			if entry.metric.ScoreBasisPoints == nil || *entry.metric.ScoreBasisPoints < 0 || *entry.metric.ScoreBasisPoints > 10000 {
				return fmt.Errorf("applicable metric %s requires a score between 0 and 10000", entry.name)
			}
		} else if entry.metric.ScoreBasisPoints != nil {
			return fmt.Errorf("non-applicable metric %s must use null score_basis_points", entry.name)
		}
	}
	for label, value := range map[string]string{
		"parser role configuration":      attempt.RoleConfiguration.Parser,
		"distiller role configuration":   attempt.RoleConfiguration.Distiller,
		"navigator role configuration":   attempt.RoleConfiguration.Navigator,
		"synthesizer role configuration": attempt.RoleConfiguration.Synthesizer,
	} {
		if err := validatePublicIdentifier(label, value); err != nil {
			return err
		}
	}
	if attempt.IneligibilityReasons == nil || attempt.Failures == nil || attempt.Exclusions == nil {
		return fmt.Errorf("ineligibility reasons, failures, and exclusions must be present")
	}
	if attempt.Eligible && len(attempt.IneligibilityReasons) != 0 {
		return fmt.Errorf("eligible attempt cannot have ineligibility reasons")
	}
	if !attempt.Eligible && len(attempt.IneligibilityReasons) == 0 {
		return fmt.Errorf("ineligible attempt requires at least one reason")
	}
	if err := validateRealAttemptAnnotations("ineligibility_reasons", attempt.IneligibilityReasons); err != nil {
		return err
	}
	if err := validateRealAttemptAnnotations("failures", attempt.Failures); err != nil {
		return err
	}
	if err := validateRealAttemptAnnotations("exclusions", attempt.Exclusions); err != nil {
		return err
	}
	return nil
}

func aoLoreComposite(attempt AOLoreAttempt, weights AOLoreMetricWeights) (int, error) {
	numerator := 0
	denominator := 0
	for _, entry := range aoLoreMetricEntries(attempt.Metrics, weights) {
		if !entry.metric.Applicable || entry.weight == 0 {
			continue
		}
		if entry.metric.ScoreBasisPoints == nil {
			return 0, fmt.Errorf("applicable metric %s has no score", entry.name)
		}
		numerator += *entry.metric.ScoreBasisPoints * entry.weight
		denominator += entry.weight
	}
	if denominator == 0 {
		return 0, fmt.Errorf("AO Lore attempt has no applicable weighted metrics")
	}
	return numerator / denominator, nil
}

func validateAOLorePair(format string, baseline, challenger AOLoreAttempt) error {
	if baseline.Format != format || challenger.Format != format ||
		baseline.FixtureCorpusSHA256 != challenger.FixtureCorpusSHA256 ||
		baseline.WorkloadProfileSHA256 != challenger.WorkloadProfileSHA256 ||
		baseline.SourceInputSHA256 != challenger.SourceInputSHA256 {
		return fmt.Errorf("AO Lore attempts are not a matched pair")
	}
	if baseline.SystemID == challenger.SystemID {
		return fmt.Errorf("baseline and challenger system_id must differ")
	}
	left := aoLoreMetricEntries(baseline.Metrics, AOLoreMetricWeights{})
	right := aoLoreMetricEntries(challenger.Metrics, AOLoreMetricWeights{})
	for i := range left {
		if left[i].metric.Applicable != right[i].metric.Applicable {
			return fmt.Errorf("matched metric applicability differs for %s", left[i].name)
		}
	}
	return nil
}

func aoLoreOutcome(attempt AOLoreAttempt, weights AOLoreMetricWeights) (AOLoreAttemptOutcome, error) {
	raw, err := aoLoreComposite(attempt, weights)
	if err != nil {
		return AOLoreAttemptOutcome{}, err
	}
	comparison := raw
	if !attempt.Eligible || !attempt.TraceIntegrity || len(attempt.Failures) > 0 {
		comparison = 0
	}
	return AOLoreAttemptOutcome{
		AttemptID: attempt.AttemptID, SystemID: attempt.SystemID, ResultSHA256: attempt.ResultSHA256,
		Metrics: attempt.Metrics, RoleConfiguration: attempt.RoleConfiguration, Eligible: attempt.Eligible,
		IneligibilityReasons: append([]string{}, attempt.IneligibilityReasons...),
		Failures:             append([]string{}, attempt.Failures...), Exclusions: append([]string{}, attempt.Exclusions...),
		TraceIntegrity: attempt.TraceIntegrity, RawCompositeScore: raw, ComparisonScore: comparison,
	}, nil
}

func buildAOLoreComparison(manifest AOLoreEvaluationManifest, attempts map[string]AOLoreAttempt) (AOLoreComparison, error) {
	report := AOLoreComparison{
		SchemaVersion: "ao.arena.ao-lore-comparison.v0.1", EvaluationID: manifest.EvaluationID,
		MetricWeightsBasisPoints: manifest.MetricWeightsBasisPoints,
		Pairs:                    make([]AOLorePairComparison, 0, len(manifest.Pairs)), Authority: "evaluation-evidence-only",
	}
	for i, reference := range manifest.Pairs {
		baseline := attempts[reference.Baseline.Path]
		challenger := attempts[reference.Challenger.Path]
		if err := validateAOLorePair(reference.Format, baseline, challenger); err != nil {
			return AOLoreComparison{}, fmt.Errorf("pair %d: %w", i, err)
		}
		if i == 0 {
			report.BaselineSystemID = baseline.SystemID
			report.ChallengerSystemID = challenger.SystemID
		} else if baseline.SystemID != report.BaselineSystemID || challenger.SystemID != report.ChallengerSystemID {
			return AOLoreComparison{}, fmt.Errorf("system roles must remain stable across AO Lore pairs")
		}
		baselineOutcome, err := aoLoreOutcome(baseline, manifest.MetricWeightsBasisPoints)
		if err != nil {
			return AOLoreComparison{}, fmt.Errorf("pair %d baseline: %w", i, err)
		}
		challengerOutcome, err := aoLoreOutcome(challenger, manifest.MetricWeightsBasisPoints)
		if err != nil {
			return AOLoreComparison{}, fmt.Errorf("pair %d challenger: %w", i, err)
		}
		winner, result := "tie", "tie"
		if baselineOutcome.ComparisonScore > challengerOutcome.ComparisonScore {
			winner, result = baseline.SystemID, "baseline_win"
		} else if challengerOutcome.ComparisonScore > baselineOutcome.ComparisonScore {
			winner, result = challenger.SystemID, "challenger_win"
		}
		report.Pairs = append(report.Pairs, AOLorePairComparison{
			Format: reference.Format, FixtureCorpusSHA256: baseline.FixtureCorpusSHA256,
			WorkloadProfileSHA256: baseline.WorkloadProfileSHA256, SourceInputSHA256: baseline.SourceInputSHA256,
			Baseline: baselineOutcome, Challenger: challengerOutcome, Winner: winner, Result: result,
		})
		report.BaselineTotal += baselineOutcome.ComparisonScore
		report.ChallengerTotal += challengerOutcome.ComparisonScore
	}
	sort.SliceStable(report.Pairs, func(i, j int) bool { return report.Pairs[i].Format < report.Pairs[j].Format })
	switch {
	case report.BaselineTotal > report.ChallengerTotal:
		report.Winner, report.Result = report.BaselineSystemID, "win"
	case report.ChallengerTotal > report.BaselineTotal:
		report.Winner, report.Result = report.ChallengerSystemID, "win"
	default:
		report.Winner, report.Result = "tie", "tie"
	}
	return report, nil
}
