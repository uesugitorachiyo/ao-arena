package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAOLoreCompareRejectsMissingAndLooseFlags(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"compare", "ao-lore", "--out", "out.json"}, want: "usage:"},
		{args: []string{"compare", "ao-lore", "--input", "in.json", "--out", "out.json", "--live"}, want: "unknown flag"},
		{args: []string{"compare", "ao-lore", "--input", "one.json", "--input", "two.json", "--out", "out.json"}, want: "duplicate --input"},
	}
	for _, tc := range tests {
		var stdout, stderr bytes.Buffer
		if code := Run(tc.args, &stdout, &stderr); code == 0 || !strings.Contains(stderr.String(), tc.want) {
			t.Fatalf("Run(%v) code=%d stderr=%q, want %q", tc.args, code, stderr.String(), tc.want)
		}
	}
}

func TestHelpDocumentsAOLoreSuppliedFileEvaluation(t *testing.T) {
	var stdout bytes.Buffer
	if code := Run([]string{"--help"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "compare ao-lore --input <manifest.json> --out <comparison.json>") {
		t.Fatalf("help missing AO Lore command:\n%s", stdout.String())
	}
}
