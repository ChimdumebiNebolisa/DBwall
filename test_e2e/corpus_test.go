package test_e2e

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/ChimdumebiNebolisa/DBwall/internal/analyzer"
	"github.com/ChimdumebiNebolisa/DBwall/internal/parser"
	"github.com/ChimdumebiNebolisa/DBwall/internal/policy"
)

type corpusCase struct {
	ID            string         `json:"id"`
	Category      string         `json:"category"`
	SQL           string         `json:"sql"`
	Policy        *policy.Policy `json:"policy,omitempty"`
	Decision      string         `json:"decision"`
	ExpectedRules []string       `json:"expected_rules"`
}

func TestAdversarialCorpus(t *testing.T) {
	data, err := os.ReadFile("testdata/corpus.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var cases []corpusCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			p := tc.Policy
			if p == nil {
				p = policy.DefaultPolicy()
			}
			stmts, err := parser.Parse(tc.SQL)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			res := analyzer.Analyze(stmts, p)
			if string(res.Decision) != tc.Decision {
				t.Fatalf("decision want %s, got %s", tc.Decision, res.Decision)
			}
			var got []string
			for _, st := range res.Statements {
				for _, f := range st.Findings {
					if !slices.Contains(got, f.Rule) {
						got = append(got, f.Rule)
					}
				}
			}
			for _, expected := range tc.ExpectedRules {
				if !slices.Contains(got, expected) {
					t.Fatalf("expected rule %s not found in %#v", expected, got)
				}
			}
		})
	}
}
