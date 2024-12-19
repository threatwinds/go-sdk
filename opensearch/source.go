package opensearch

import (
	"encoding/json"
	gosdk "github.com/threatwinds/go-sdk"
)

// ParseSource parses the HitSource object into a JSON string and then decode it into the provided destination object.
// The destination object must be a pointer to the desired type.
func (h HitSource) ParseSource(dest interface{}) error {
	j, err := json.Marshal(h)
	if err != nil {
		return gosdk.Error("failed to encode HitSource", err, nil)
	}

	err = json.Unmarshal(j, dest)
	if err != nil {
		return gosdk.Error("failed to decode HitSource", err, nil)
	}

	return nil
}

// SetSource sets the HitSource object from the provided source object.
func (h *HitSource) SetSource(src interface{}) error {
	j, err := json.Marshal(src)
	if err != nil {
		return gosdk.Error("failed to encode source object", err, nil)
	}

	err = json.Unmarshal(j, h)
	if err != nil {
		return gosdk.Error("failed to decode source object", err, nil)
	}

	return nil
}
