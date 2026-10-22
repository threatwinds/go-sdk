package plugins

import "testing"

func TestEventFromRecord(t *testing.T) {
	cases := []struct {
		name     string
		record   string
		wantCode uint32
		wantErr  bool
	}{
		{"empty status code", `{"id":"a","dataType":"linux","statusCode":""}`, 0, false},
		{"numeric text status code", `{"id":"a","statusCode":"404"}`, 404, false},
		{"numeric status code", `{"id":"a","statusCode":404}`, 404, false},
		{"status code that is not a number", `{"id":"a","statusCode":"N/A"}`, 0, false},
		{"no status code", `{"id":"a"}`, 0, false},
		{"not an object", `[1,2]`, 0, true},
	}
	for _, c := range cases {
		e, err := EventFromRecord([]byte(c.record))
		if (err != nil) != c.wantErr {
			t.Fatalf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
		}
		if err != nil {
			continue
		}
		if e.StatusCode != c.wantCode {
			t.Errorf("%s: statusCode = %d, want %d", c.name, e.StatusCode, c.wantCode)
		}
		if e.Id != "a" {
			t.Errorf("%s: id = %q, want a", c.name, e.Id)
		}
	}
}

func TestEventFromRecordKeepsTheRestOfARealRow(t *testing.T) {
	row := `{"tenantId":"t","@timestamp":"2026-10-22 06:31:02.382","dataType":"linux","id":"x","protocol":"","statusCode":"","errors":[],"origin":{"host":"h","process":"systemd"},"target":{},"log":{"codeLine":"79","n":1}}`
	e, err := EventFromRecord([]byte(row))
	if err != nil {
		t.Fatal(err)
	}
	if e.Timestamp != "2026-10-22 06:31:02.382" || e.Origin.GetHost() != "h" || e.Log["codeLine"].GetStringValue() != "79" {
		t.Errorf("fields lost: %+v", e)
	}
}
