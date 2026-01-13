package os

import (
	"encoding/json"

	"github.com/threatwinds/go-sdk/utils"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ParseSource parses the HitSource object into a JSON string and then unmarshals it into the provided destination object.
// The destination object must be a pointer to the desired type.
func (h *HitSource) ParseSource(dest interface{}) error {
	j, err := json.Marshal(h)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, dest)
}

// ParseSourceToProtoMessage serializes the HitSource to JSON and unmarshals it into the provided ProtoMessage.
func (h *HitSource) ParseSourceToProtoMessage(dest protoreflect.ProtoMessage) error {
	j, err := json.Marshal(h)
	if err != nil {
		return err
	}

	return utils.StringToProtoMessage(utils.PointerOf(string(j)), dest)
}

// SetSource sets the HitSource object from the provided source object.
func (h *HitSource) SetSource(src interface{}) error {
	j, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, h)
}

// SetSourceFromProtoMessage sets the HitSource object by deserializing a ProtoMessage into its map representation.
func (h *HitSource) SetSourceFromProtoMessage(src protoreflect.ProtoMessage) error {
	j, err := utils.ProtoMessageToBytes(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(*j, h)
}
