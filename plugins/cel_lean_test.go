package plugins

import (
	"testing"

	"github.com/threatwinds/go-sdk/jsondoc"
)

const leanDoc = `{
	"dataType": "firewall-sophos-xg",
	"raw": "<30>device=\"SFW\" log_type=\"Firewall\"",
	"origin": {"ip": "192.168.1.5", "port": 443},
	"target": {"ip": "8.8.8.8"},
	"log": {"type": "Event", "subType": "Authentication", "component": "SSL VPN", "count": "7", "empty": "", "nested": {"a": {"b": "x"}}}
}`

func TestEvaluateJSONMatchesEvaluate(t *testing.T) {
	expressions := []string{
		`exists("log.type")`,
		`!exists("log.missing")`,
		`equals("log.type","Event") && equals("log.subType","Authentication")`,
		`oneOf("log.component",["SSL VPN","VPN"]) || equals("origin.port", 80)`,
		`equals("origin.port", 443)`,
		`startsWith("raw","<30>") && contains("raw","log_type")`,
		`regexMatch("log.type","^Ev")`,
		`inCIDR("origin.ip","192.168.0.0/16") && !inCIDR("target.ip","10.0.0.0/8")`,
		`greaterThan("log.count", 5) && lessOrEqual("log.count", 7)`,
		`equals("log.nested.a.b","x")`,
		`exists("log.empty") && equals("log.empty","")`,
		`int(safe("log.count","0")) > 3`,
		`double(safe("log.count","0")) == 7.0`,
		`true`,
		`false`,
		`log.type == "Event"`,
		`dataType == "firewall-sophos-xg" && exists("log.type")`,
		`exists("log.type"`,
		`unknownFunction("x")`,
	}

	for _, expr := range expressions {
		doc := leanDoc
		full, errFull := NewCELCache("test").Evaluate(&doc, expr)
		lean, errLean := NewCELCache("test").EvaluateJSON(&doc, expr)

		if (errFull == nil) != (errLean == nil) {
			t.Errorf("%s: error mismatch, full=%v lean=%v", expr, errFull, errLean)
			continue
		}
		if full != lean {
			t.Errorf("%s: full=%v lean=%v", expr, full, lean)
		}
	}
}

func TestEvaluateJSONReusesCompiledProgram(t *testing.T) {
	c := NewCELCache("test")
	doc := leanDoc
	expr := `equals("log.type","Event")`

	for i := 0; i < 3; i++ {
		got, err := c.EvaluateJSON(&doc, expr)
		if err != nil || !got {
			t.Fatalf("iteration %d: got=%v err=%v", i, got, err)
		}
	}

	if n := c.leanSize.Load(); n != 1 {
		t.Fatalf("expected a single cached expression, got %d", n)
	}
}

func TestEvaluateJSONFallsBackWhenVariablesAreUsed(t *testing.T) {
	c := NewCELCache("test")
	doc := leanDoc

	got, err := c.EvaluateJSON(&doc, `dataType == "firewall-sophos-xg"`)
	if err != nil || !got {
		t.Fatalf("variable expression: got=%v err=%v", got, err)
	}

	v, _ := c.lean.Load(`dataType == "firewall-sophos-xg"`)
	if v.(*leanEntry).prg != nil {
		t.Fatal("an expression that needs event variables must not use the lean program")
	}
}

func TestEvaluateDocMatchesEvaluate(t *testing.T) {
	expressions := []string{
		`exists("log.type")`, `!exists("log.missing")`,
		`equals("log.type","Event") && equals("log.subType","Authentication")`,
		`oneOf("log.component",["SSL VPN","VPN"]) || equals("origin.port", 80)`,
		`equals("origin.port", 443)`, `startsWith("raw","<30>") && contains("raw","log_type")`,
		`regexMatch("log.type","^Ev")`, `inCIDR("origin.ip","192.168.0.0/16") && !inCIDR("target.ip","10.0.0.0/8")`,
		`greaterThan("log.count", 5) && lessOrEqual("log.count", 7)`, `equals("log.nested.a.b","x")`,
		`exists("log.empty") && equals("log.empty","")`, `int(safe("log.count","0")) > 3`,
		`double(safe("log.count","0")) == 7.0`, `contains("log.nested","x")`, `exists("log.nested.#")`,
		`log.type == "Event"`, `exists("log.type"`,
	}

	for _, expr := range expressions {
		text := leanDoc
		doc, err := jsondoc.Parse(text)
		if err != nil {
			t.Fatal(err)
		}

		want, errWant := NewCELCache("test").Evaluate(&text, expr)
		got, errGot := NewCELCache("test").EvaluateDoc(doc, expr)

		if (errWant == nil) != (errGot == nil) {
			t.Errorf("%s: error mismatch, text=%v doc=%v", expr, errWant, errGot)
			continue
		}
		if want != got {
			t.Errorf("%s: text=%v doc=%v", expr, want, got)
		}
	}
}
