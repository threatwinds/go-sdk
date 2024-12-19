package go_sdk

import (
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
//   - *error: A pointer to a error if an error occurs, otherwise nil.
func ToString(object protoreflect.ProtoMessage) (*string, error) {
	if object == nil {
		return nil, Error(Trace(), map[string]interface{}{
			"error": "nil input parameter",
		})
	}

	objectBytes, err := protojson.Marshal(object)
	if err != nil {
		return nil, Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to parse object",
		})
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
		return Error(Trace(), map[string]interface{}{
			"error": "nil input parameter",
		})
	}

	if len(*str) > maxMessageSize {
		return Error(Trace(), map[string]interface{}{
			"error": "message size exceeds the limit",
		})
	}

	err := protojson.Unmarshal([]byte(*str), object)
	if err != nil {
		return Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "failed to parse object",
		})
	}

	return nil
}
