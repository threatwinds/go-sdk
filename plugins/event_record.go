package plugins

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/threatwinds/go-sdk/utils"
)

// EventFromRecord parses a stored log record into an Event.
//
// A record read back from the store is not an Event as it was sent: the table
// keeps statusCode as text and fills it with "" when the source had none, while
// the Event field is a uint32 that protojson refuses to read from "". Left
// alone that fails the whole record, so every record of a typical source — one
// that never sets a status code — could not be turned back into an Event at
// all. A statusCode that is not a number is dropped for the same reason: it
// carries nothing the Event could hold.
func EventFromRecord(raw []byte) (*Event, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("event record is not a JSON object: %w", err)
	}

	if v, ok := fields["statusCode"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil {
			if _, err := strconv.ParseUint(s, 10, 32); err != nil {
				delete(fields, "statusCode")
			}
		}
	}

	cleaned, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	str := string(cleaned)
	var e Event
	if err := utils.StringToProtoMessage(&str, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
