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
