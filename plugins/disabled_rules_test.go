package plugins

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
)

func TestSplitDisabledRulesLoadsBothSpellings(t *testing.T) {
	cases := []struct {
		name      string
		doc       string
		wantIDs   []uint64
		wantNames []string
	}{
		{"v11 numeric ids", `{"disabledRules":[1,2,3]}`, []uint64{1, 2, 3}, nil},
		{"v12 file names", `{"disabledRules":["linux/ssh.yml","a.yml"]}`, nil, []string{"linux/ssh.yml", "a.yml"}},
		{"mixed", `{"disabledRules":[7,"a.yml","8"]}`, []uint64{7, 8}, []string{"a.yml"}},
		{"explicit names key", `{"disabledRuleNames":["x.yml"]}`, nil, []string{"x.yml"}},
	}
	for _, c := range cases {
		var cfg Config
		if err := protojson.Unmarshal(splitDisabledRules([]byte(c.doc)), &cfg); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(cfg.DisabledRules) != len(c.wantIDs) || len(cfg.DisabledRuleNames) != len(c.wantNames) {
			t.Errorf("%s: ids=%v names=%v", c.name, cfg.DisabledRules, cfg.DisabledRuleNames)
		}
	}
}

func TestSplitDisabledRulesAppliesToEachTenant(t *testing.T) {
	var cfg Config
	doc := `{"tenants":[{"id":"t","disabledRules":["a.yml"]},{"id":"u","disabledRules":[5]}]}`
	if err := protojson.Unmarshal(splitDisabledRules([]byte(doc)), &cfg); err != nil {
		t.Fatal(err)
	}
	if got := cfg.Tenants[0].DisabledRuleNames; len(got) != 1 || got[0] != "a.yml" {
		t.Errorf("tenant t names = %v", got)
	}
	if got := cfg.Tenants[1].DisabledRules; len(got) != 1 || got[0] != 5 {
		t.Errorf("tenant u ids = %v", got)
	}
}
