package utils

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxMessageSize = 10 * 1024 * 1024 // 10MB limit

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

	objectString := string(objectBytes)

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

	err := unmarshaler.Unmarshal([]byte(*str), object)
	if err != nil {
		return fmt.Errorf("failed to parse object: %w", err)
	}

	return nil
}
