package utils

import (
	"errors"
	"fmt"
	"github.com/threatwinds/go-sdk/catcher"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxMessageSize = 10 * 1024 * 1024 // 10MB limit

// ToString converts a given ProtoMessage object to its JSON string representation.
// It returns a pointer to the JSON string and a pointer to an error if any error occurs during marshaling.
//
// Parameters:
//   - object: The ProtoMessage object to be converted.
//
// Returns:
//   - *string: A pointer to the JSON string representation of the object.
//   - *error: A pointer to an error if an error occurs, otherwise nil.
func ToString(object protoreflect.ProtoMessage) (*string, error) {
	if object == nil {
		return nil, catcher.Error("cannot convert to string", errors.New("object is a nil pointer"), nil)
	}

	objectBytes, err := protojson.Marshal(object)
	if err != nil {
		return nil, catcher.Error("cannot convert to string", err, nil)
	}

	objectString := string(objectBytes)

	return &objectString, nil
}

// ToObject parses a JSON-encoded string into a given ProtoMessage object.
//
// Parameters:
//   - str: A pointer to the JSON-encoded string.
//   - object: The ProtoMessage object to unmarshal the JSON string into.
//
// Returns:
//   - error: An error object if the unmarshalling fails, otherwise nil.
func ToObject(str *string, object protoreflect.ProtoMessage) error {
	if str == nil || object == nil {
		return catcher.Error("cannot convert to object", errors.New("object or string is a nil pointer"), map[string]any{
			"nilStr":    str == nil,
			"nilObject": object == nil,
		})
	}

	if len(*str) > maxMessageSize {
		return catcher.Error("cannot convert to object", errors.New("message size exceeds limit"), map[string]any{
			"size":  fmt.Sprintf("%d bytes", len(*str)),
			"limit": fmt.Sprintf("%d bytes", maxMessageSize),
		})
	}

	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}

	err := unmarshaler.Unmarshal([]byte(*str), object)
	if err != nil {
		return catcher.Error("failed to parse object", err, nil)
	}

	return nil
}
