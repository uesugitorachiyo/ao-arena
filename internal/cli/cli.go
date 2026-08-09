package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ao-foundry/ao-arena/internal/arena"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	var err error
	switch args[0] {
	case "suite":
		err = runSuite(args[1:], stdout)
	case "competitor":
		err = runCompetitor(args[1:], stdout)
	case "run":
		err = runRunner(args[1:], stdout)
	case "evidence":
		err = runEvidence(args[1:], stdout)
	case "score":
		err = runScore(args[1:], stdout)
	case "compare":
		err = runCompare(args[1:], stdout)
	case "report":
		err = runReport(args[1:], stdout)
	case "gate":
		err = runGate(args[1:], stdout)
	case "safety":
		err = runSafety(args[1:], stdout)
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}

	if err != nil {
		fmt.Fprintf(stderr, "arena: %v\n", err)
		return 1
	}
	return 0
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `AO Arena benchmark CLI

Commands:
  suite validate --suite <path>
  competitor validate --competitor <path>
  run fixture --suite <path> --competitor <path> --out <dir>
  evidence validate --bundle <path>
  score --attempt <path> --scorecard <path> --out <path>
  compare --suite <path> --fixture-mode --out <path>
  compare real-attempts --input <manifest.json> --out <comparison.json>
  compare ao-lore --input <manifest.json> --out <comparison.json>
  report render --report <json> --out <markdown>
  gate promotion --report <json> --out <json>
  safety scan --path <path> --out <json>`)
}

func runSuite(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "validate" {
		return fmt.Errorf("usage: arena suite validate --suite <path>")
	}
	path := flagValue(args[1:], "--suite")
	if path == "" {
		return fmt.Errorf("missing --suite")
	}
	if _, err := arena.LoadAndValidateSuite(path); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "suite valid")
	return nil
}

func runCompetitor(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "validate" {
		return fmt.Errorf("usage: arena competitor validate --competitor <path>")
	}
	path := flagValue(args[1:], "--competitor")
	if path == "" {
		return fmt.Errorf("missing --competitor")
	}
	if _, err := arena.LoadAndValidateCompetitor(path); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "competitor valid")
	return nil
}

func runRunner(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "fixture" {
		return fmt.Errorf("usage: arena run fixture --suite <path> --competitor <path> --out <dir>")
	}
	suite := flagValue(args[1:], "--suite")
	competitor := flagValue(args[1:], "--competitor")
	out := flagValue(args[1:], "--out")
	if suite == "" || competitor == "" || out == "" {
		return fmt.Errorf("missing --suite, --competitor, or --out")
	}
	count, err := arena.RunFixture(suite, competitor, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "fixture attempts written: %d\n", count)
	return nil
}

func runEvidence(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "validate" {
		return fmt.Errorf("usage: arena evidence validate --bundle <path>")
	}
	path := flagValue(args[1:], "--bundle")
	if path == "" {
		return fmt.Errorf("missing --bundle")
	}
	if err := arena.ValidateEvidenceBundle(path); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "evidence bundle valid")
	return nil
}

func runScore(args []string, stdout io.Writer) error {
	attemptPath := flagValue(args, "--attempt")
	scorecardPath := flagValue(args, "--scorecard")
	out := flagValue(args, "--out")
	if attemptPath == "" || scorecardPath == "" || out == "" {
		return fmt.Errorf("usage: arena score --attempt <path> --scorecard <path> --out <path>")
	}
	score, err := arena.ScoreAttempt(attemptPath, scorecardPath, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "score written: %d\n", score)
	return nil
}

func runCompare(args []string, stdout io.Writer) error {
	if len(args) > 0 && args[0] == "real-attempts" {
		return runCompareRealAttempts(args[1:], stdout)
	}
	if len(args) > 0 && args[0] == "ao-lore" {
		return runCompareAOLore(args[1:], stdout)
	}
	suite := flagValue(args, "--suite")
	out := flagValue(args, "--out")
	if suite == "" || out == "" {
		return fmt.Errorf("usage: arena compare --suite <path> --fixture-mode --out <path>")
	}
	if !hasFlag(args, "--fixture-mode") {
		return fmt.Errorf("compare requires --fixture-mode in v0.1")
	}
	report, err := arena.CompareFixture(suite, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "comparison report written: %s winner=%s\n", filepath.Clean(out), report.Winner)
	return nil
}

func runCompareAOLore(args []string, stdout io.Writer) error {
	input, out, err := realAttemptComparePaths(args)
	if err != nil {
		if strings.HasPrefix(err.Error(), "usage:") {
			return fmt.Errorf("usage: arena compare ao-lore --input <manifest.json> --out <comparison.json>")
		}
		return err
	}
	report, err := arena.CompareAOLore(input, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "AO Lore comparison written: pairs=%d winner=%s result=%s\n", len(report.Pairs), report.Winner, report.Result)
	return nil
}

func runCompareRealAttempts(args []string, stdout io.Writer) error {
	input, out, err := realAttemptComparePaths(args)
	if err != nil {
		return err
	}
	report, err := arena.CompareRealAttempts(input, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "real-attempt comparison written: pairs=%d winner=%s result=%s\n", report.PairCount, report.Winner, report.Result)
	return nil
}

func realAttemptComparePaths(args []string) (string, string, error) {
	var input, out string
	for i := 0; i < len(args); {
		name := args[i]
		if name != "--input" && name != "--out" {
			return "", "", fmt.Errorf("unknown flag or argument %q", name)
		}
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			return "", "", fmt.Errorf("usage: arena compare real-attempts --input <manifest.json> --out <comparison.json>")
		}
		value := args[i+1]
		switch name {
		case "--input":
			if input != "" {
				return "", "", fmt.Errorf("duplicate --input")
			}
			input = value
		case "--out":
			if out != "" {
				return "", "", fmt.Errorf("duplicate --out")
			}
			out = value
		}
		i += 2
	}
	if input == "" || out == "" {
		return "", "", fmt.Errorf("usage: arena compare real-attempts --input <manifest.json> --out <comparison.json>")
	}
	return input, out, nil
}

func runReport(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "render" {
		return fmt.Errorf("usage: arena report render --report <json> --out <markdown>")
	}
	reportPath := flagValue(args[1:], "--report")
	out := flagValue(args[1:], "--out")
	if reportPath == "" || out == "" {
		return fmt.Errorf("missing --report or --out")
	}
	if err := arena.RenderReport(reportPath, out); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "markdown report written: %s\n", filepath.Clean(out))
	return nil
}

func runGate(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "promotion" {
		return fmt.Errorf("usage: arena gate promotion --report <json> --out <json>")
	}
	reportPath := flagValue(args[1:], "--report")
	out := flagValue(args[1:], "--out")
	if reportPath == "" || out == "" {
		return fmt.Errorf("missing --report or --out")
	}
	gate, err := arena.WritePromotionGate(reportPath, out)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "promotion gate written: %s status=%s\n", filepath.Clean(out), gate.Status)
	return nil
}

func runSafety(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "scan" {
		return fmt.Errorf("usage: arena safety scan --path <path> --out <json>")
	}
	path := flagValue(args[1:], "--path")
	out := flagValue(args[1:], "--out")
	if path == "" || out == "" {
		return fmt.Errorf("missing --path or --out")
	}
	report, err := arena.WriteSafetyScan(path, out)
	if err != nil {
		return err
	}
	if report.Status != "passed" {
		return fmt.Errorf("safety scan failed with %d findings", report.FindingCount)
	}
	fmt.Fprintf(stdout, "safety scan written: %s status=%s\n", filepath.Clean(out), report.Status)
	return nil
}

func flagValue(args []string, name string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}
