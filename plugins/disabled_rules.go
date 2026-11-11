package plugins

import (
	"encoding/json"
	"strconv"
)

// splitDisabledRules keeps both spellings of the disabledRules key loading.
//
// v11 lists rules by numeric id under disabledRules; v12 lists them by file
// name under the same key. The two cannot share one typed field, so names move
// to disabledRuleNames before the document reaches protojson. Anything that
// parses as a number stays where it is.
func splitDisabledRules(doc []byte) []byte {
	var root map[string]any
	if err := json.Unmarshal(doc, &root); err != nil {
		return doc
	}

	changed := splitDisabledIn(root)
	if tenants, ok := root["tenants"].([]any); ok {
		for _, t := range tenants {
			if m, ok := t.(map[string]any); ok && splitDisabledIn(m) {
				changed = true
			}
		}
	}
	if !changed {
		return doc
	}

	out, err := json.Marshal(root)
	if err != nil {
		return doc
	}
	return out
}

func splitDisabledIn(m map[string]any) bool {
	list, ok := m["disabledRules"].([]any)
	if !ok {
		return false
	}

	var ids, names []any
	for _, v := range list {
		switch x := v.(type) {
		case string:
			if _, err := strconv.ParseUint(x, 10, 64); err == nil {
				ids = append(ids, x)
			} else {
				names = append(names, x)
			}
		default:
			ids = append(ids, v)
		}
	}
	if len(names) == 0 {
		return false
	}

	if len(ids) == 0 {
		delete(m, "disabledRules")
	} else {
		m["disabledRules"] = ids
	}
	if existing, ok := m["disabledRuleNames"].([]any); ok {
		names = append(existing, names...)
	}
	m["disabledRuleNames"] = names
	return true
}
