package plugins

import (
	"testing"

	sdkutils "github.com/threatwinds/go-sdk/utils"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/types/known/structpb"
)

func valOf(t *testing.T, s string) *structpb.Value {
	t.Helper()
	v, err := structpb.NewValue(s)
	if err != nil {
		t.Fatalf("NewValue(%q): %v", s, err)
	}
	return v
}

// TestEventMarshalUsesEventKey confirms Event.Event serializes under "event"
// (not "log") in the JSON handed to CEL and OpenSearch.
func TestEventMarshalUsesEventKey(t *testing.T) {
	e := &Event{
		Id:        "1",
		Timestamp: "2026-09-30T00:00:00Z",
		DataType:  "wineventlog",
		Event:     map[string]*structpb.Value{"eventCode": valOf(t, "4624"), "severity": valOf(t, "high")},
		Controls:  []string{"NIST-800-53-AC-3"},
	}

	s, err := sdkutils.ProtoMessageToString(e)
	if err != nil {
		t.Fatalf("ProtoMessageToString: %v", err)
	}
	if !gjson.Get(*s, "event.eventCode").Exists() {
		t.Fatalf("expected event.eventCode in %s", *s)
	}
	if gjson.Get(*s, "log").Exists() {
		t.Fatalf("event should NOT still serialize under 'log': %s", *s)
	}
	if len(e.Controls) != 1 || e.Controls[0] != "NIST-800-53-AC-3" {
		t.Fatalf("controls not populated: %v", e.Controls)
	}
	if !gjson.Get(*s, "controls.0").Exists() || gjson.Get(*s, "controls.0").String() != "NIST-800-53-AC-3" {
		t.Fatalf("controls not serialized to JSON: %s", *s)
	}
}

// TestEventRoundTrip confirms a doc serialized with "event" reads back into an
// Event (the correlation path: OS _source -> StringToProtoMessage).
func TestEventRoundTrip(t *testing.T) {
	s, err := sdkutils.ProtoMessageToString(&Event{
		Id: "2", DataType: "wineventlog",
		Event: map[string]*structpb.Value{"eventCode": valOf(t, "4624")},
	})
	if err != nil {
		t.Fatalf("ProtoMessageToString: %v", err)
	}
	var back Event
	if err := sdkutils.StringToProtoMessage(s, &back); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !gjson.Get(*s, "event.eventCode").Exists() || back.Event == nil {
		t.Fatalf("round-trip lost event: %s / %+v", *s, back.Event)
	}
	if got := back.Event["eventCode"].GetStringValue(); got != "4624" {
		t.Fatalf("round-trip lost eventCode value: got %q", got)
	}
	if back.Id != "2" {
		t.Fatalf("round-trip lost id: got %q", back.Id)
	}
}

// TestControlsOmittedWhenEmpty confirms an empty controls list is absent from
// the JSON so the field costs nothing today.
func TestControlsOmittedWhenEmpty(t *testing.T) {
	s, err := sdkutils.ProtoMessageToString(&Event{Id: "3", DataType: "x"})
	if err != nil {
		t.Fatalf("ProtoMessageToString: %v", err)
	}
	if gjson.Get(*s, "controls").Exists() {
		t.Fatalf("empty controls should be omitted: %s", *s)
	}
}
