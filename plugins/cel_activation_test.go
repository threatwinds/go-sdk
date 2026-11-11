package plugins

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// realisticEventJSON is a ~4KB event shaped like a parsed UTMStack event (top-level
// scalar fields + nested object + array). It exercises the same code paths the live
// cel.plugin sees (a proto.Message serializes to exactly this shape).
const realisticEventJSON = `{
  "id": "b7f2c1d0-0000-4000-8000-000000000001",
  "timestamp": "2026-11-04T10:00:00Z",
  "dataType": "aws",
  "dataSource": "AWS CloudTrail - Quantfall",
  "tenantId": "ce66672c-e36d-4761-a8c8-90058fee1a24",
  "action": "AssumeRole",
  "actionResult": "success",
  "origin": {"ip": "203.0.113.10", "user": "arn:aws:iam::123456789012:user/bob", "host": "signin.amazonaws.com"},
  "target": {"domain": "sts.amazonaws.com"},
  "event": {
    "eventSource": "sts.amazonaws.com",
    "eventName": "AssumeRole",
    "eventVersion": "1.09",
    "awsRegion": "us-east-1",
    "userAgent": "aws-cli/2.13.0",
    "userIdentity": {"type": "IAMUser", "accountId": "123456789012", "arn": "arn:aws:iam::123456789012:user/bob"},
    "responseElements": {"assumedRoleUser": {"arn": "arn:aws:sts::123456789012:assumed-role/DeployRole/sess"}}
  },
  "utm": {"awsRegion": "us-east-1", "eventType": "AwsApiCall"},
  "raw": "{\"source\":\"raw-payload\"}"
}`

// ruleExpressions returns N distinct, realistic rule `where` expressions (string-keyed
// overloads + bare identifiers), like the 81 aws rules the cel.plugin evaluates per event.
func ruleExpressions(n int) []string {
	exprs := make([]string, 0, n)
	srcs := []string{"sts.amazonaws.com", "iam.amazonaws.com", "ec2.amazonaws.com", "s3.amazonaws.com", "signin.amazonaws.com", "ssm.amazonaws.com"}
	names := []string{"AssumeRole", "CreateUser", "AttachRolePolicy", "GetSecretValue", "ConsoleLogin", "DeleteDetector"}
	for i := 0; i < n; i++ {
		s := srcs[i%len(srcs)]
		e := names[i%len(names)]
		switch i % 6 {
		case 0:
			exprs = append(exprs, fmt.Sprintf(`equals("event.eventSource","%s") && equals("actionResult","success")`, s))
		case 1:
			exprs = append(exprs, fmt.Sprintf(`equals("event.eventSource","%s") && oneOf("action",["%s","ConsoleLogin"])`, s, e))
		case 2:
			exprs = append(exprs, fmt.Sprintf(`regexMatch("event.eventName","^%s") && exists("origin.ip")`, e))
		case 3:
			exprs = append(exprs, fmt.Sprintf(`contains("event.userAgent","aws") && inCIDR("origin.ip","203.0.113.0/24") && !equals("event.eventSource","%s")`, s))
		case 4:
			exprs = append(exprs, fmt.Sprintf(`greaterOrEqual("utm.awsRegion","a") && lessThan("event.eventVersion","2.0") && exists("event.awsRegion")`))
		case 5:
			// bare top-level identifiers to prove valuesMap variables are still declared
			exprs = append(exprs, `actionResult == "success" && dataType == "aws" && action == "AssumeRole"`)
		}
	}
	return exprs
}

// TestEvalActivationParity is the behavior gate: evaluating an expression via the
// prepare-once API must return the exact same result as the current per-call Eval,
// for string data, map data, and the string-keyed / bare-identifier overloads alike.
func TestEvalActivationParity(t *testing.T) {
	cache := NewCELCache("parity_test")
	exprs := []string{
		`equals("action","AssumeRole")`,
		`equals("action","DeleteDetector")`,
		`exists("origin.ip")`,
		`exists("target.missing")`,
		`greaterThan("utm.awsRegion","a")`,
		`oneOf("actionResult",["success","failed"])`,
		`contains("event.userAgent","cli")`,
		`inCIDR("origin.ip","203.0.113.0/24")`,
		`regexMatch("event.eventName","^Assume")`,
		`actionResult == "success" && dataType == "aws"`, // bare identifiers
		`1/0`,                                              // compile/runtime error case
	}

	// reference: current per-call Eval over a JSON string
	jsonData := realisticEventJSON
	for _, expr := range exprs {
		refRes, refErr := cache.Eval(expr, &jsonData)

		act, err := cache.PrepareActivation(&jsonData)
		assert.NoError(t, err, "PrepareActivation must not error for valid JSON")
		if err != nil {
			t.Fatalf("PrepareActivation error: %v", err)
		}
		res, err := cache.EvalWithActivation(expr, act)
		assert.Equal(t, refErr != nil, err != nil, "err parity for %q", expr)
		if refErr == nil && err == nil {
			assert.Equal(t, refRes, res, "result parity for %q", expr)
		}
	}

	// also over a map, to cover the map branch
	var m map[string]any
	assert.NoError(t, json.Unmarshal([]byte(realisticEventJSON), &m))
	for _, expr := range []string{`equals("action","AssumeRole")`, `actionResult == "success" && dataType == "aws"`, `exists("target.missing")`} {
		refRes, refErr := cache.Eval(expr, m)
		act, err := cache.PrepareActivation(m)
		assert.NoError(t, err)
		res, err2 := cache.EvalWithActivation(expr, act)
		assert.Equal(t, refErr != nil, err2 != nil)
		if refErr == nil && err2 == nil {
			assert.Equal(t, refRes, res)
		}
	}
}

// BenchmarkCelRuleLoop measures the cel.plugin hot loop (N rules x one event) two ways:
//   - per_rule:   the current behavior, Eval(expr, event) re-derives the activation each rule
//   - prepare_on: prepare the activation once per event, then EvalWithActivation per rule
// The ratio is the Fix-1 win.
func BenchmarkCelRuleLoop(b *testing.B) {
	const nRules = 81
	exprs := ruleExpressions(nRules)
	jsonData := realisticEventJSON
	cache := NewCELCache("bench_loop")

	// warm the program cache so neither sub-benchmark pays compile cost
	for _, e := range exprs {
		_, _ = cache.Eval(e, &jsonData)
	}

	b.ResetTimer()
	b.Run("per_rule", func(bb *testing.B) {
		for bb.Loop() {
			for _, e := range exprs {
				_, _ = cache.Eval(e, &jsonData)
			}
		}
	})

	b.Run("prepare_once", func(bb *testing.B) {
		for bb.Loop() {
			act, err := cache.PrepareActivation(&jsonData)
			if err != nil {
				bb.Fatalf("prepare: %v", err)
			}
			for _, e := range exprs {
				_, _ = cache.EvalWithActivation(e, act)
			}
		}
	})
}
