package plugins

import (
	"testing"

	"github.com/threatwinds/go-sdk/utils"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStringToProtoMessageKeepsParsedFieldsFromALegacyLogKey(t *testing.T) {
	for _, key := range []string{"log", "event"} {
		doc := `{"id":"x","` + key + `":{"user":"root"}}`
		var e Event
		if err := utils.StringToProtoMessage(&doc, &e); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if e.Event["user"].GetStringValue() != "root" {
			t.Errorf("%s: parsed fields were dropped: %+v", key, e.Event)
		}
	}

	both := `{"log":{"a":"old"},"event":{"a":"new"}}`
	var e Event
	if err := utils.StringToProtoMessage(&both, &e); err != nil {
		t.Fatal(err)
	}
	if e.Event["a"].GetStringValue() != "new" {
		t.Errorf("event must win over log, got %+v", e.Event)
	}
}

func TestStringToProtoMessageLeavesOtherMessagesAlone(t *testing.T) {
	doc := `{"id":"x","log":"not an event field"}`
	var l Log
	if err := utils.StringToProtoMessage(&doc, &l); err != nil {
		t.Fatalf("Log must still load: %v", err)
	}
}

func TestEventKeyIsChosenByTheProcessAndReadBothWays(t *testing.T) {
	t.Cleanup(func() { _ = utils.SetEventKey("event") })

	e := &Event{Id: "x", Event: map[string]*structpb.Value{"user": structpb.NewStringValue("root")}}
	for _, key := range []string{"event", "log"} {
		if err := utils.SetEventKey(key); err != nil {
			t.Fatal(err)
		}
		out, err := utils.ProtoMessageToString(e)
		if err != nil {
			t.Fatal(err)
		}
		if !gjson.Get(*out, key+".user").Exists() {
			t.Errorf("key %q: not serialized under it: %s", key, *out)
		}
		other := map[string]string{"event": "log", "log": "event"}[key]
		if gjson.Get(*out, other).Exists() {
			t.Errorf("key %q: %q must not also be written: %s", key, other, *out)
		}

		var back Event
		if err := utils.StringToProtoMessage(out, &back); err != nil || back.Event["user"].GetStringValue() != "root" {
			t.Errorf("key %q: round trip lost the fields (err=%v): %+v", key, err, back.Event)
		}
	}
	if err := utils.SetEventKey("other"); err == nil {
		t.Error("an unknown key must be rejected")
	}
}

func alertWithEvents(t *testing.T) *Alert {
	t.Helper()
	ev := func(id, user string) *Event {
		return &Event{Id: id, Event: map[string]*structpb.Value{"user": structpb.NewStringValue(user)}}
	}
	return &Alert{Id: "a1", Name: "n", LastEvent: ev("e2", "root"), Events: []*Event{ev("e1", "alice"), ev("e2", "root")}}
}

func TestEventKeyAppliesToEventsNestedInAnAlert(t *testing.T) {
	t.Cleanup(func() { _ = utils.SetEventKey("event") })

	for _, key := range []string{"event", "log"} {
		if err := utils.SetEventKey(key); err != nil {
			t.Fatal(err)
		}
		out, err := utils.ProtoMessageToString(alertWithEvents(t))
		if err != nil {
			t.Fatal(err)
		}
		other := map[string]string{"event": "log", "log": "event"}[key]

		for _, path := range []string{"lastEvent", "events.0", "events.1"} {
			if gjson.Get(*out, path+"."+key+".user").String() == "" {
				t.Errorf("key %q: %s is not serialized under it: %s", key, path, *out)
			}
			if gjson.Get(*out, path+"."+other).Exists() {
				t.Errorf("key %q: %s must not also carry %q: %s", key, path, other, *out)
			}
		}

		var back Alert
		if err := utils.StringToProtoMessage(out, &back); err != nil {
			t.Fatalf("key %q: %v", key, err)
		}
		if back.LastEvent.GetEvent()["user"].GetStringValue() != "root" || len(back.Events) != 2 ||
			back.Events[0].GetEvent()["user"].GetStringValue() != "alice" {
			t.Errorf("key %q: round trip lost the nested events: %+v", key, &back)
		}
	}
}

func TestAnAlertDocumentWithTheLegacyLogKeyStillLoads(t *testing.T) {
	doc := `{"id":"a1","lastEvent":{"id":"e","log":{"user":"root"}},"events":[{"id":"e","log":{"user":"alice"}}]}`
	var a Alert
	if err := utils.StringToProtoMessage(&doc, &a); err != nil {
		t.Fatal(err)
	}
	if a.LastEvent.GetEvent()["user"].GetStringValue() != "root" || a.Events[0].GetEvent()["user"].GetStringValue() != "alice" {
		t.Errorf("nested parsed fields were dropped: %+v", &a)
	}
}

func TestRewritingNestedEventsKeepsLargeNumbersExact(t *testing.T) {
	t.Cleanup(func() { _ = utils.SetEventKey("event") })
	if err := utils.SetEventKey("log"); err != nil {
		t.Fatal(err)
	}
	a := &Alert{Id: "a", LastEvent: &Event{Id: "e", Event: map[string]*structpb.Value{"n": structpb.NewNumberValue(1234567890123)}}}
	out, err := utils.ProtoMessageToString(a)
	if err != nil {
		t.Fatal(err)
	}
	if got := gjson.Get(*out, "lastEvent.log.n").Raw; got != "1234567890123" {
		t.Errorf("number rewritten as %s", got)
	}
}
