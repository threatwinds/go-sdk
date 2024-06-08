package opensearch

import "encoding/json"

// ParseSource parses the HitSource object into a JSON string and then unmarshals it into the provided destination object.
// The destination object must be a pointer to the desired type.
func (h HitSource) ParseSource(dest interface{}) error {
	j, err := json.Marshal(h)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, dest)
}

// SetSource sets the HitSource object from the provided source object.
func (h *HitSource) SetSource(src interface{}) error {
	j, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(j, h)
}
