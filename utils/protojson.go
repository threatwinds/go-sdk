package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxMessageSize = 10 * 1024 * 1024 // 10MB limit

// eventKey is the JSON key an Event carries its parsed fields under. The two
// UTMStack lines store them under different names ("event" on v11, "log" on
// v12), and a document is serialized with exactly one.
var eventKey atomic.Pointer[string]

// EventKeyEnv is the environment variable a process reads at startup to choose
// the event key, so that a deployment decides it once for every process that
// runs in it: "event" for a v11 installation, "log" for a v12 one.
const EventKeyEnv = "UTM_EVENT_KEY"

func init() {
	if err := setEventKeyFromEnv(os.Getenv); err != nil {
		// A key that is wrong must not fall back to the default: the process
		// would start and write every event's parsed fields where nothing reads
		// them.
		panic(err)
	}
}

func setEventKeyFromEnv(getenv func(string) string) error {
	v := getenv(EventKeyEnv)
	if v == "" {
		return nil
	}
	if err := SetEventKey(v); err != nil {
		return fmt.Errorf("%s: %w", EventKeyEnv, err)
	}
	return nil
}

// EventKey is the key Event messages are serialized with.
func EventKey() string {
	if k := eventKey.Load(); k != nil {
		return *k
	}
	return "event"
}

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

// eventMessage is the full name of the message whose parsed-fields object this
// package renames.
const eventMessage protoreflect.FullName = "plugins.Event"

var containsEventCache sync.Map // protoreflect.FullName -> bool

// containsEvent reports whether a message is an Event or holds one at any depth.
func containsEvent(md protoreflect.MessageDescriptor) bool {
	return containsEventSeen(md, map[protoreflect.FullName]bool{})
}

func containsEventSeen(md protoreflect.MessageDescriptor, seen map[protoreflect.FullName]bool) bool {
	name := md.FullName()
	if v, ok := containsEventCache.Load(name); ok {
		return v.(bool)
	}
	if name == eventMessage {
		return true
	}
	if seen[name] {
		return false
	}
	seen[name] = true

	found := false
	fields := md.Fields()
	for i := 0; i < fields.Len() && !found; i++ {
		fd := fields.Get(i)
		switch {
		case fd.IsMap():
			if vm := fd.MapValue().Message(); vm != nil {
				found = containsEventSeen(vm, seen)
			}
		case fd.Message() != nil:
			found = containsEventSeen(fd.Message(), seen)
		}
	}
	// A negative answer is only final for the message the walk started from:
	// partway through a cycle it may have been cut short.
	if found || len(seen) == 1 {
		containsEventCache.Store(name, found)
	}
	return found
}

// withEventKey renames the key protojson wrote for Event.event, wherever an
// Event sits in the message: at the top, or nested as in an Alert's lastEvent
// and events.
func withEventKey(out []byte, object protoreflect.ProtoMessage) []byte {
	k := eventKey.Load()
	if k == nil || *k == "event" {
		return out
	}
	md := object.ProtoReflect().Descriptor()
	if !containsEvent(md) {
		return out
	}

	// An Event itself is the hot path. protojson writes fields in number order
	// and every field before this one is a scalar, so the first "event": is the
	// field and never text inside a value, whose quotes are escaped.
	if md.FullName() == eventMessage {
		return bytes.Replace(out, []byte(`"event":`), []byte(`"`+*k+`":`), 1)
	}
	return rewriteEventKeys(out, md, "event", *k)
}

// rewriteEventKeys renames, in every Event object of a document shaped like md,
// the key from to to. A document it cannot parse is returned as it came.
func rewriteEventKeys(doc []byte, md protoreflect.MessageDescriptor, from, to string) []byte {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return doc
	}
	renameInNode(root, md, from, to)

	out, err := json.Marshal(root)
	if err != nil {
		return doc
	}
	return out
}

func renameInNode(node any, md protoreflect.MessageDescriptor, from, to string) {
	obj, ok := node.(map[string]any)
	if !ok {
		return
	}
	if md.FullName() == eventMessage {
		if v, ok := obj[from]; ok {
			if _, has := obj[to]; !has {
				obj[to] = v
			}
			delete(obj, from)
		}
	}

	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		var child protoreflect.MessageDescriptor
		isMap := fd.IsMap()
		switch {
		case isMap:
			child = fd.MapValue().Message()
		default:
			child = fd.Message()
		}
		if child == nil || !containsEvent(child) {
			continue
		}

		val, ok := obj[fd.JSONName()]
		if !ok {
			val, ok = obj[string(fd.Name())]
		}
		if !ok {
			continue
		}

		switch v := val.(type) {
		case []any:
			for _, el := range v {
				renameInNode(el, child, from, to)
			}
		case map[string]any:
			if isMap {
				for _, el := range v {
					renameInNode(el, child, from, to)
				}
			} else {
				renameInNode(v, child, from, to)
			}
		}
	}
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

// legacyEventKey lets a document that still carries an Event's parsed fields
// under "log" load into the field that is now called "event", wherever the Event
// sits in it. DiscardUnknown would otherwise drop them without a word.
func legacyEventKey(doc string, object protoreflect.ProtoMessage) string {
	if !strings.Contains(doc, `"log"`) {
		return doc
	}
	md := object.ProtoReflect().Descriptor()
	if !containsEvent(md) {
		return doc
	}
	return string(rewriteEventKeys([]byte(doc), md, "log", "event"))
}
