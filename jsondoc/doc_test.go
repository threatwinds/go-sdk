package jsondoc

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var words = []string{"log", "origin", "target", "ip", "type", "subType", "user", "id", "tags", "count", "a", "b", "c_d", "x-y", "k1", "Name"}

func randString(r *rand.Rand) string {
	pool := []string{"", "Event", "192.168.1.5", "with \"quotes\"", "line\nbreak", "tab\t", "ünïcode ✓", "<html>&amp;", "back\\slash", "12", "0.5", "true", "null"}
	return pool[r.Intn(len(pool))]
}

func randValue(r *rand.Rand, depth int) any {
	switch n := r.Intn(9); {
	case n < 3:
		return randString(r)
	case n == 3:
		return json.Number(fmt.Sprint(r.Intn(100000)))
	case n == 4:
		return json.Number(fmt.Sprintf("%.2f", r.Float64()*1000))
	case n == 5:
		return r.Intn(2) == 0
	case n == 6:
		return nil
	case n == 7 && depth < 3:
		return randObject(r, depth+1)
	case depth < 3:
		arr := make([]any, r.Intn(4))
		for i := range arr {
			arr[i] = randValue(r, depth+1)
		}
		return arr
	}
	return randString(r)
}

func randObject(r *rand.Rand, depth int) map[string]any {
	m := map[string]any{}
	for i := 0; i < 1+r.Intn(5); i++ {
		m[words[r.Intn(len(words))]] = randValue(r, depth)
	}
	return m
}

func paths(prefix string, v any, out *[]string) {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			*out = append(*out, p)
			paths(p, c, out)
		}
	case []any:
		for i, c := range x {
			p := fmt.Sprintf("%s.%d", prefix, i)
			*out = append(*out, p)
			paths(p, c, out)
		}
	}
}

func canon(t *testing.T, raw string) any {
	t.Helper()
	var v any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("bad json %q: %v", raw, err)
	}
	return v
}

func sameResult(t *testing.T, ctx string, got, want gjson.Result) {
	t.Helper()
	if got.Exists() != want.Exists() || got.Type != want.Type {
		t.Fatalf("%s: exists/type got (%v,%v) want (%v,%v)", ctx, got.Exists(), got.Type, want.Exists(), want.Type)
	}
	if got.Type == gjson.JSON {
		if !reflect.DeepEqual(canon(t, got.Raw), canon(t, want.Raw)) {
			t.Fatalf("%s: json got %s want %s", ctx, got.Raw, want.Raw)
		}
		return
	}
	if got.String() != want.String() || got.Float() != want.Float() || got.Int() != want.Int() || got.Bool() != want.Bool() {
		t.Fatalf("%s: got %q/%v want %q/%v", ctx, got.String(), got.Float(), want.String(), want.Float())
	}
}

func TestGetMatchesGjson(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	checked := 0

	for i := 0; i < 1500; i++ {
		root := randObject(r, 0)
		text, _ := json.Marshal(root)

		d, err := Parse(string(text))
		if err != nil {
			t.Fatal(err)
		}

		var ps []string
		paths("", root, &ps)
		ps = append(ps, "missing", "log.missing", "a.b.c.d.e", "log.0", "tags.99", "-", "log.", ".log", "log..ip")
		sort.Strings(ps)

		for _, p := range ps {
			sameResult(t, fmt.Sprintf("doc %d path %q", i, p), d.Get(p), gjson.Get(string(text), p))

			if d.Has(p) != gjson.Get(string(text), p).Exists() {
				t.Fatalf("doc %d path %q: Has disagrees with gjson", i, p)
			}
			checked++
		}
	}

	t.Logf("%d path lookups compared with gjson", checked)
}

func TestRichPathsFallBackToGjson(t *testing.T) {
	text := `{"log":{"items":[{"n":"a"},{"n":"b"}],"name":"x","other":"y"}}`
	d, _ := Parse(text)

	for _, p := range []string{"log.items.#", "log.items.#.n", "log.*", "log.na?e", `log.items.#(n=="b").n`, "log.items|@reverse", "log.name|@this", `log\.x`} {
		sameResult(t, p, d.Get(p), gjson.Get(text, p))
	}
}

func TestSetAndDeleteMatchSjson(t *testing.T) {
	r := rand.New(rand.NewSource(2))

	for i := 0; i < 600; i++ {
		root := randObject(r, 0)
		for k, v := range root {
			if _, isArr := v.([]any); isArr {
				delete(root, k)
			}
		}

		text, _ := json.Marshal(root)
		d, _ := Parse(string(text))
		want := string(text)

		for op := 0; op < 12; op++ {
			depth := 1 + r.Intn(3)
			segs := make([]string, depth)
			for j := range segs {
				segs[j] = words[r.Intn(len(words))]
			}
			path := strings.Join(segs, ".")

			if r.Intn(3) == 0 {
				d.Delete(path)
				want, _ = sjson.Delete(want, path)
				continue
			}

			var v any
			switch r.Intn(4) {
			case 0:
				v = randString(r)
			case 1:
				v = int64(r.Intn(1000))
			case 2:
				v = r.Float64() * 100
			default:
				v = r.Intn(2) == 0
			}

			var existing gjson.Result = gjson.Get(want, path)
			parent := gjson.Get(want, strings.Join(segs[:depth-1], "."))
			if depth > 1 && parent.Exists() && !parent.IsObject() {
				continue
			}
			_ = existing

			if err := d.Set(path, v); err != nil {
				t.Fatal(err)
			}
			var err error
			if want, err = sjson.Set(want, path, v); err != nil {
				t.Fatal(err)
			}
		}

		if !reflect.DeepEqual(canon(t, d.String()), canon(t, want)) {
			t.Fatalf("doc %d diverged\n got  %s\n want %s", i, d.String(), want)
		}
	}
}

func TestRoundTripAndEscaping(t *testing.T) {
	text := `{"raw":"a<b>&c \"q\" é\n","n":1.50,"big":12345678901234567890,"o":{"k":[1,2,{"z":null}]}}`
	d, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}

	if d.GetString("raw") != "a<b>&c \"q\" é\n" {
		t.Fatalf("raw: %q", d.GetString("raw"))
	}

	if d.Get("big").Raw != "12345678901234567890" || d.Get("n").Raw != "1.50" {
		t.Fatalf("numbers lost their text: %q %q", d.Get("big").Raw, d.Get("n").Raw)
	}

	if err := d.Set("o.new", "x"); err != nil {
		t.Fatal(err)
	}

	back, err := Parse(d.String())
	if err != nil || back.GetString("o.new") != "x" || back.Get("o.k.2.z").Type != gjson.Null {
		t.Fatalf("round trip failed: %v %s", err, d.String())
	}

	if strings.Contains(d.String(), "\\"+"u003c") || !strings.Contains(d.String(), "a<b>&c") {
		t.Fatalf("HTML characters must not be escaped, to match sjson output: %s", d.String())
	}
}

func TestSetValueFromStruct(t *testing.T) {
	type geo struct {
		City string  `json:"city,omitempty"`
		Lat  float64 `json:"latitude,omitempty"`
		Asn  uint64  `json:"asn,omitempty"`
	}

	d, _ := Parse(`{}`)
	if err := d.Set("origin.geolocation", &geo{City: "Seattle", Lat: 47.6}); err != nil {
		t.Fatal(err)
	}

	want, _ := sjson.Set(`{}`, "origin.geolocation", &geo{City: "Seattle", Lat: 47.6})
	if !reflect.DeepEqual(canon(t, d.String()), canon(t, want)) {
		t.Fatalf("got %s want %s", d.String(), want)
	}

	if d.Get("origin.geolocation.latitude").Float() != 47.6 || d.Has("origin.geolocation.asn") {
		t.Fatal("omitempty fields must be absent")
	}
}

func TestRichPathsInSetAndDeleteMatchSjson(t *testing.T) {
	for _, path := range []string{"a..b", "log.", ".x", "a.b.", "log.items.#", `a\.b`, "x.y|@this"} {
		text := `{"log":{"items":[1,2]},"a":{"b":1}}`
		d, _ := Parse(text)
		want, wantErr := sjson.Set(text, path, "v")
		gotErr := d.Set(path, "v")

		if (wantErr == nil) != (gotErr == nil) {
			t.Fatalf("set %q: error mismatch want=%v got=%v", path, wantErr, gotErr)
		}
		if wantErr == nil && !reflect.DeepEqual(canon(t, d.String()), canon(t, want)) {
			t.Fatalf("set %q: got %s want %s", path, d.String(), want)
		}

		d2, _ := Parse(text)
		d2.Delete(path)
		wantDel, _ := sjson.Delete(text, path)
		if !reflect.DeepEqual(canon(t, d2.String()), canon(t, wantDel)) {
			t.Fatalf("delete %q: got %s want %s", path, d2.String(), wantDel)
		}
	}
}
