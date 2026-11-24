// Package jsondoc holds a JSON object in memory so a pipeline can read and change it without
// scanning or copying its text on every operation.
//
// Reads return gjson results and paths follow gjson's syntax, so code written against gjson works
// unchanged. Plain dotted paths are answered straight from the parsed tree; any path that uses
// gjson's richer syntax (wildcards, queries, modifiers, escapes) is answered by gjson over the
// document's text, which is produced once and kept until the next change.
package jsondoc

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Doc is a JSON object. It is not safe for concurrent use.
type Doc struct {
	root map[string]any
	text string
	ok   bool
}

// Parse reads a JSON object. Numbers keep their original text.
func Parse(s string) (*Doc, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()

	var root map[string]any
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}

	if root == nil {
		root = map[string]any{}
	}

	return &Doc{root: root, text: s, ok: true}, nil
}

// String returns the document as JSON.
func (d *Doc) String() string {
	if d.ok {
		return d.text
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(d.root)

	d.text, d.ok = strings.TrimRight(buf.String(), "\n"), true

	return d.text
}

func isSimplePath(path string) bool {
	if path == "" {
		return false
	}

	segment := false
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case c == '.':
			if !segment {
				return false
			}
			segment = false
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_', c == '-':
			segment = true
		default:
			return false
		}
	}

	return segment
}

func step(cur any, seg string) (any, bool) {
	switch v := cur.(type) {
	case map[string]any:
		x, ok := v[seg]
		return x, ok
	case []any:
		i, err := strconv.Atoi(seg)
		if err != nil || i < 0 || i >= len(v) {
			return nil, false
		}
		return v[i], true
	}

	return nil, false
}

func (d *Doc) lookup(path string) (any, bool) {
	var cur any = d.root
	for len(path) > 0 {
		seg := path
		if i := strings.IndexByte(path, '.'); i >= 0 {
			seg, path = path[:i], path[i+1:]
		} else {
			path = ""
		}

		var ok bool
		if cur, ok = step(cur, seg); !ok {
			return nil, false
		}
	}

	return cur, true
}

// Has reports whether path exists.
func (d *Doc) Has(path string) bool {
	if !isSimplePath(path) {
		return gjson.Get(d.String(), path).Exists()
	}

	_, ok := d.lookup(path)

	return ok
}

// Get returns the value at path as a gjson result.
func (d *Doc) Get(path string) gjson.Result {
	if !isSimplePath(path) {
		return gjson.Get(d.String(), path)
	}

	v, ok := d.lookup(path)
	if !ok {
		return gjson.Result{}
	}

	return toResult(v)
}

// GetString returns the value at path as text, like gjson's Result.String.
func (d *Doc) GetString(path string) string {
	return d.Get(path).String()
}

func toResult(v any) gjson.Result {
	switch x := v.(type) {
	case nil:
		return gjson.Result{Type: gjson.Null, Raw: "null"}
	case string:
		return gjson.Result{Type: gjson.String, Str: x}
	case bool:
		if x {
			return gjson.Result{Type: gjson.True, Raw: "true"}
		}
		return gjson.Result{Type: gjson.False, Raw: "false"}
	case json.Number:
		f, _ := x.Float64()
		return gjson.Result{Type: gjson.Number, Num: f, Raw: x.String()}
	case float64:
		return gjson.Result{Type: gjson.Number, Num: x, Raw: strconv.FormatFloat(x, 'f', -1, 64)}
	case int:
		return gjson.Result{Type: gjson.Number, Num: float64(x), Raw: strconv.Itoa(x)}
	case int64:
		return gjson.Result{Type: gjson.Number, Num: float64(x), Raw: strconv.FormatInt(x, 10)}
	case uint64:
		return gjson.Result{Type: gjson.Number, Num: float64(x), Raw: strconv.FormatUint(x, 10)}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)

	return gjson.Result{Type: gjson.JSON, Raw: strings.TrimRight(buf.String(), "\n")}
}

// normalize turns any value into what Parse would have produced for its JSON form, so a value set
// from a struct or a typed number reads back the same way as one that came from the text.
func normalize(v any) (any, error) {
	switch x := v.(type) {
	case nil, string, bool, json.Number, float64, int, int64, uint64, map[string]any, []any:
		return x, nil
	}

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// viaText applies an edit that needs sjson's path syntax to the document's text and reloads it.
func (d *Doc) viaText(edit func(string) (string, error)) error {
	out, err := edit(d.String())
	if err != nil {
		return err
	}

	next, err := Parse(out)
	if err != nil {
		return err
	}

	*d = *next

	return nil
}

// Set stores v at path, creating the objects on the way.
func (d *Doc) Set(path string, v any) error {
	val, err := normalize(v)
	if err != nil {
		return err
	}

	if !isSimplePath(path) {
		return d.viaText(func(text string) (string, error) { return sjson.Set(text, path, val) })
	}

	cur := d.root
	for {
		i := strings.IndexByte(path, '.')
		if i < 0 {
			cur[path] = val
			break
		}

		seg := path[:i]
		path = path[i+1:]

		next, ok := cur[seg].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[seg] = next
		}

		cur = next
	}

	d.ok = false

	return nil
}

// Delete removes path if it exists.
func (d *Doc) Delete(path string) {
	if !isSimplePath(path) {
		_ = d.viaText(func(text string) (string, error) { return sjson.Delete(text, path) })
		return
	}

	cur := d.root
	for {
		i := strings.IndexByte(path, '.')
		if i < 0 {
			if _, ok := cur[path]; ok {
				delete(cur, path)
				d.ok = false
			}
			return
		}

		next, ok := cur[path[:i]].(map[string]any)
		if !ok {
			return
		}

		cur, path = next, path[i+1:]
	}
}
