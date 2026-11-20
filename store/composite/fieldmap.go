package composite

import (
	"encoding/json"
	"strings"
)

// PrefixMap returns the two field-name translations a Config takes, for stores
// that differ only in what they call the object holding the parsed fields: the
// current store says "log.user" where the secondary says "event.user". Names that
// do not start with the prefix pass through unchanged, and so does the prefix
// alone.
func PrefixMap(primary, secondary string) (toSecondary, fromSecondary func(string) string) {
	swap := func(from, to string) func(string) string {
		return func(field string) string {
			if field == from {
				return to
			}
			if rest, ok := strings.CutPrefix(field, from+"."); ok {
				return to + "." + rest
			}
			return field
		}
	}
	return swap(primary, secondary), swap(secondary, primary)
}

// RenameDocKey returns a Config.SecondaryDocument that renames one top-level key of
// a record. A record without the key, or that is not a JSON object, is returned
// as it came, and a key already present under the new name is left alone.
func RenameDocKey(from, to string) func(json.RawMessage) json.RawMessage {
	return func(doc json.RawMessage) json.RawMessage {
		var top map[string]json.RawMessage
		if err := json.Unmarshal(doc, &top); err != nil {
			return doc
		}
		v, ok := top[from]
		if !ok {
			return doc
		}
		if _, exists := top[to]; !exists {
			top[to] = v
		}
		delete(top, from)

		out, err := json.Marshal(top)
		if err != nil {
			return doc
		}
		return out
	}
}
