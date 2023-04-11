package core

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	admissionV1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Parse constructs OPA policy from string
func Parse(content, ruleQuery string) (Policy, error) {
	// validate module
	module, err := ast.ParseModule("", content)
	if err != nil {
		return Policy{}, err
	}

	if module == nil {
		return Policy{}, fmt.Errorf("Failed to parse module: empty content")
	}

	var valid bool
	for _, rule := range module.Rules {
		if rule.Head.Name == ast.Var(ruleQuery) {
			valid = true
			break
		}
	}

	if !valid {
		return Policy{}, fmt.Errorf("rule `%s` is not found", ruleQuery)
	}

	policy := Policy{
		module: module,
		pkg:    strings.Split(module.Package.String(), "package ")[1],
	}

	return policy, nil
}

// Eval validates data against given policy
// returns error if there're any violations found
func (p Policy) Eval(data interface{}, query string) error {
	rego := rego.New(
		rego.Query(fmt.Sprintf("data.%s.%s", p.pkg, query)),
		rego.ParsedModule(p.module),
		rego.Input(data),
	)

	// Run evaluation.
	rs, err := rego.Eval(context.Background())
	if err != nil {
		return err
	}
	for _, r := range rs {
		for _, expr := range r.Expressions {
			switch reflect.TypeOf(expr.Value).Kind() {
			case reflect.Slice:
				s := expr.Value.([]interface{})
				if len(s) > 0 {
					err := NoValidError{
						Details: s,
					}
					return err
				}
			case reflect.Map:
				s := expr.Value.(map[string]interface{})
				err := NoValidError{
					Details: s,
				}
				return err
			case reflect.String:
				s := expr.Value.(string)
				err := NoValidError{
					Details: s,
				}
				return err
			}
		}
	}
	return nil
}

// EvalGateKeeperCompliant modifies the data to be Gatekeeper compliant and validates data against given policy
// returns error if there're any violations found
func (p Policy) EvalGateKeeperCompliant(data map[string]interface{}, parameters map[string]interface{}, query string) error {

	obj := unstructured.Unstructured{
		Object: data,
	}

	bytesData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req := admissionV1.AdmissionRequest{
		Name: obj.GetName(),
		Kind: metav1.GroupVersionKind{
			Kind:    obj.GetObjectKind().GroupVersionKind().Kind,
			Version: obj.GetObjectKind().GroupVersionKind().Version,
			Group:   obj.GetObjectKind().GroupVersionKind().Group,
		},
		Object: runtime.RawExtension{
			Raw: bytesData,
		},
	}
	input := map[string]interface{}{"review": req, "parameters": parameters}

	return p.Eval(input, query)
}
