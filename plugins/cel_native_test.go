package plugins

import (
	"testing"

	"github.com/threatwinds/go-sdk/jsondoc"
)

var nativeDocs = []string{
	`{"log":{"type":"Event","count":"7","n":7,"f":7.5,"yes":true,"no":false,"nul":null,"empty":"","arr":["a","b"],"obj":{"k":"v"},"s":"  Mixed Case "},"origin":{"port":443,"ip":"1.2.3.4"},"raw":"<30>device=SFW log_type=Firewall"}`,
	`{"log":{"type":"event","count":"abc","n":"8","f":"x","yes":"true","arr":[],"obj":{}},"origin":{"port":"443"}}`,
	`{}`,
	`{"log":{"type":5,"count":9007199254740991,"n":-1}}`,
}

var nativeExprs = []string{
	`true`,
	`inCIDR("origin.ip","1.2.0.0/16")`,
	`inCIDR("origin.ip","::/0")`,
	`inCIDR("origin.ip","bad")`,
	`inCIDR("log.type","0.0.0.0/0") || inCIDR("origin.ip","0.0.0.0/0")`,
	`exists("log.type")`,
	`!exists("log.nul")`,
	`equals("log.type","Event")`,
	`equals("log.n", 7)`,
	`equals("log.n", "7")`,
	`equals("log.f", 7.5)`,
	`equals("log.yes", true)`,
	`equals("log.yes", "true")`,
	`equals("log.count", 7)`,
	`equals("origin.port", 443) || equals("origin.port", 80)`,
	`oneOf("log.type", ["Event","event"])`,
	`oneOf("origin.port", [80, 443, 8080])`,
	`oneOf("log.yes", [true, "x"])`,
	`!oneOf("log.type", ["Event"]) && exists("log.type")`,
	`equalsIgnoreCase("log.type","EVENT")`,
	`contains("log.s","Mixed")`,
	`contains("log.arr","a")`,
	`contains("log.obj",["v","zzz"])`,
	`contains("raw", [" log_type ", "nope"])`,
	`containsAll("raw", ["device", " log_type"])`,
	`startsWith("raw","<30>")`,
	`startsWith("raw", ["<31>", " <30>"])`,
	`endsWith("raw","Firewall")`,
	`endsWith("log.s", ["Case ", "x"])`,
	`regexMatch("log.count","^[0-9]+$")`,
	`regexMatch("log.obj",".+")`,
	`regexMatch("log.type","(")`,
	`greaterOrEqual("log.count",0) && lessOrEqual("log.count",9007199254740991)`,
	`greaterThan("log.n", 5)`,
	`lessThan("log.n", 8.5)`,
	`lessOrEqual("log.n", "7")`,
	`true && !exists("log.zzz") && (exists("log.type") && !oneOf("log.type",["x","y"]))`,
	`exists("a") || exists("b") || equals("log.type","Event")`,
}

func TestNativePredicateMatchesCEL(t *testing.T) {
	c := NewCELCache("native_test")

	for _, expr := range nativeExprs {
		if _, ok := compileNative(expr); !ok {
			t.Errorf("%s: expected to compile natively", expr)
		}

		for _, raw := range nativeDocs {
			doc, err := jsondoc.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}

			got, err := c.EvaluateDoc(doc, expr)
			if err != nil {
				t.Fatalf("%s: %v", expr, err)
			}

			text := raw
			want, err := c.Evaluate(&text, expr)
			if err != nil {
				t.Fatalf("%s: %v", expr, err)
			}

			if got != want {
				t.Errorf("%s on %s: native %v, cel %v", expr, raw, got, want)
			}
		}
	}
}

func TestNativeLeavesUnsupportedToCEL(t *testing.T) {
	for _, expr := range []string{
		`int(safe("log.count","0")) > 3`,
		`equals("log.type", "a" + "b")`,
		`log.type == "Event"`,
		`isWeekend("log.when")`,
	} {
		if _, ok := compileNative(expr); ok {
			t.Errorf("%s: should not compile natively", expr)
		}
	}
}
