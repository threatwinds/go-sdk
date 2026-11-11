package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxMessageSize = 10 * 1024 * 1024 // 10MB limit

// eventKey is the JSON key an Event carries its parsed fields under. The two
// UTMStack lines store them under different names ("event" on v11, "log" on
// v12), and a document is serialized with exactly one.
var eventKey atomic.Pointer[string]

// SetEventKey chooses the JSON key Event messages are serialized with: "event"
// (the default) or "log". Reading accepts both whatever is chosen. Call it once
// at startup; any other value is rejected.
func SetEventKey(key string) error {
	if key != "event" && key != "log" {
		return fmt.Errorf("event key must be %q or %q, got %q", "event", "log", key)
	}
	eventKey.Store(&key)
	return nil
}

// withEventKey renames the key protojson wrote for Event.event. protojson
// writes fields in number order and every field before it is a scalar, so the
// first "event": in the output is the field itself and never text inside a
// value, whose quotes are escaped.
func withEventKey(out []byte, object protoreflect.ProtoMessage) []byte {
	k := eventKey.Load()
	if k == nil || *k == "event" || object.ProtoReflect().Descriptor().FullName() != "plugins.Event" {
		return out
	}
	return bytes.Replace(out, []byte(`"event":`), []byte(`"`+*k+`":`), 1)
}

// ProtoMessageToString converts a given ProtoMessage object to its JSON string representation.
// It returns a pointer to the JSON string and a pointer to an error if any error occurs during marshaling.
//
// Parameters:
//   - object: The ProtoMessage object to be converted.
//
// Returns:
//   - *string: A pointer to the JSON string representation of the object.
//   - *error: A pointer to an error if an error occurs, otherwise nil.
func ProtoMessageToString(object protoreflect.ProtoMessage) (*string, error) {
	if object == nil {
		return nil, fmt.Errorf("cannot convert to string: object is a nil pointer")
	}

	objectBytes, err := protojson.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("cannot convert to string: %w", err)
	}

	objectString := string(withEventKey(objectBytes, object))

	return &objectString, nil
}

// ProtoMessageToBytes converts a given ProtoMessage object to its binary representation.
// It returns a pointer to the binary representation and a pointer to an error if any error occurs during marshaling.
//
// Parameters:
//   - object: The ProtoMessage object to be converted.
//
// Returns:
//   - *[]byte: A pointer to the binary representation of the object.
//   - error: A pointer to an error if an error occurs, otherwise nil.
func ProtoMessageToBytes(object protoreflect.ProtoMessage) (*[]byte, error) {
	if object == nil {
		return nil, fmt.Errorf("cannot convert to bytes: object is a nil pointer")
	}

	objectBytes, err := protojson.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("cannot convert to bytes: %w", err)
	}

	objectBytes = withEventKey(objectBytes, object)

	return &objectBytes, nil
}

// StringToProtoMessage parses a JSON-encoded string into a given ProtoMessage object.
//
// Parameters:
//   - str: A pointer to the JSON-encoded string.
//   - object: The ProtoMessage object to unmarshal the JSON string into.
//
// Returns:
//   - error: An error object if the unmarshalling fails, otherwise nil.
func StringToProtoMessage(str *string, object protoreflect.ProtoMessage) error {
	if str == nil || object == nil {
		return fmt.Errorf("cannot convert to object: object or string is a nil pointer (nilStr=%v, nilObject=%v)", str == nil, object == nil)
	}

	if len(*str) > maxMessageSize {
		return fmt.Errorf("cannot convert to object: message size exceeds limit (size=%d bytes, limit=%d bytes)", len(*str), maxMessageSize)
	}

	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}

	err := unmarshaler.Unmarshal([]byte(legacyEventKey(*str, object)), object)
	if err != nil {
		return fmt.Errorf("failed to parse object: %w", err)
	}

	return nil
}

// legacyEventKey lets an Event document that still carries its parsed fields
// under "log" load into the field that is now called "event". DiscardUnknown
// would otherwise drop them without a word.
func legacyEventKey(doc string, object protoreflect.ProtoMessage) string {
	if !strings.Contains(doc, `"log"`) || object.ProtoReflect().Descriptor().FullName() != "plugins.Event" {
		return doc
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(doc), &top); err != nil {
		return doc
	}
	legacy, ok := top["log"]
	if !ok {
		return doc
	}
	if _, has := top["event"]; !has {
		top["event"] = legacy
	}
	delete(top, "log")

	out, err := json.Marshal(top)
	if err != nil {
		return doc
	}
	return string(out)
}
