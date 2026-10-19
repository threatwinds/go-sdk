package plugins

import (
	"testing"

	"github.com/tidwall/sjson"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestUtmBagRoundtrip(t *testing.T) {
	draft := `{"event":{"Workload":"Exchange","eventCode":4672},"action":"Send"}`
	
	draft, _ = sjson.Set(draft, "utm.Workload", "Exchange")
	draft, _ = sjson.Set(draft, "utm.eventCode", "4672")
	
	var e Event
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := unmarshaler.Unmarshal([]byte(draft), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	
	if len(e.Utm) != 2 {
		t.Fatalf("expected 2 utm fields, got %d: %v", len(e.Utm), e.Utm)
	}
	if e.Utm["Workload"] == nil || e.Utm["Workload"].GetStringValue() != "Exchange" {
		t.Errorf("utm.Workload = %v, want Exchange", e.Utm["Workload"])
	}
	if e.Utm["eventCode"] == nil || e.Utm["eventCode"].GetStringValue() != "4672" {
		t.Errorf("utm.eventCode = %v, want 4672", e.Utm["eventCode"])
	}
	
	if e.Event["Workload"] == nil {
		t.Error("event.Workload was lost")
	}
	if e.Event["eventCode"] == nil {
		t.Error("event.eventCode was lost")
	}
}
