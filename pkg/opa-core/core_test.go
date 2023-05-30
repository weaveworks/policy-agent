package core

import (
	"testing"
)

type testCaseParsePolicy struct {
	name     string
	content  string
	hasError bool
}

func TestParse(t *testing.T) {
	cases := []testCaseParsePolicy{
		{
			name: "single rule",
			content: `
		package core
		violation[issue] {
			issue = "test"
		}`},
		{
			name: "multiple rules at once",
			content: `
			package core
			violation[issue] {
				issue = "test"
			}
			violation[issue] {
				issue = "test"
			}
		`,
			hasError: false,
		},
		{
			name: "invalid syntax",
			content: `
			package core
			issue = "test issue")
		`,
			hasError: true,
		},
		{
			name: "invalid syntax",
			content: `
			package core
			issue = "test issue")
		`,
			hasError: true,
		},
		{
			name:     "empty content",
			content:  "",
			hasError: true,
		},
		{
			name: "policy without package",
			content: `
			violation[issue] {
				x = 3
			}
		`,
			hasError: true,
		},
	}

	for _, c := range cases {
		_, err := Parse(c.content, "violation")
		if c.hasError {
			if err == nil {
				t.Errorf("[%s]: passed but should have been failed", c.name)
			}
		} else {
			if err != nil {
				t.Errorf("[%s]: %v", c.name, err)
			}
		}
	}
}

type testCaseEval struct {
	name         string
	content      string
	violationMsg string
	hasViolation bool
}

func TestEval(t *testing.T) {
	cases := []testCaseEval{
		{
			name: "rule has no violation",
			content: `
			package core
			violation[issue] {
				1 == 2
				issue = "violation test"
			}`,
		},
		{
			name: "rule has an empty violation",
			content: `
			package core
			violation[issue] {
				issue = ""
			}`,
			violationMsg: "[\"\"]",
			hasViolation: true,
		},
		{
			name: "rule has a bool violation",
			content: `
			package core
			violation[issue] {
				issue = true
			}`,
			violationMsg: "[true]",
			hasViolation: true,
		},
	}

	for _, c := range cases {
		policy, err := Parse(c.content, "violation")
		err = policy.Eval("{}", "violation")

		if c.hasViolation {
			if err == nil {
				t.Errorf("[%s]: passed but should have been failed", c.name)
			} else if err.Error() != c.violationMsg {
				t.Errorf("[%s]: expected error msg '%s' but got %s", c.name, c.violationMsg, err)
			}
		} else {
			if err != nil {
				t.Errorf("[%s]: %v", c.name, err)
			}
		}
	}
}

func TestEvalGateKeeperCompliant(t *testing.T) {
	cases := []testCaseEval{
		{
			name: "rule has no violation",
			content: `
			package magalix.advisor.labels.missing_label

label := input.parameters.label

violation[result] {
  result=input.review.name
}
`,
			hasViolation: true,
			violationMsg: "[\"kubernetes-downwardapi-volume-example\"]",
		},
	}

	for _, c := range cases {
		policy, err := Parse(c.content, "violation")
		err = policy.EvalGateKeeperCompliant(
			map[string]interface{}{
				"apiVersion": "v1", "kind": "Pod",
				"metadata": map[string]interface{}{"name": "kubernetes-downwardapi-volume-example",
					"labels":      map[string]interface{}{"zone": "us-est-coast", "cluster": "test-cluster1", "rack": "rack-22"},
					"annotations": map[string]interface{}{"build": "two", "builder": "john-doe"}}},
			map[string]interface{}{"probe": "livenessProbe"},
			"violation",
		)

		if c.hasViolation {
			if err == nil {
				t.Errorf("[%s]: passed but should have been failed", c.name)
			} else if err.Error() != c.violationMsg {
				t.Errorf("[%s]: expected error msg '%s' but got %s", c.name, c.violationMsg, err)
			}
		} else {
			if err != nil {
				t.Errorf("[%s]: %v", c.name, err)
			}
		}
	}

}
