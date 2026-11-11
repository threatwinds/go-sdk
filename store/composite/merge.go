package composite

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/threatwinds/go-sdk/store"
)

// overfetch is how many times the requested size each store is asked for when a
// ranking has to be assembled from two of them: a value that is in the overall
// top n can sit below the top n of either store alone.
const (
	overfetch    = 4
	overfetchMin = 100
)

func widen(n int) int {
	if n <= 0 {
		return overfetchMin
	}
	if w := n * overfetch; w > overfetchMin {
		return w
	}
	return overfetchMin
}

func mergeBuckets(a, b []store.Bucket, n int) []store.Bucket {
	sum := make(map[string]int64, len(a)+len(b))
	for _, x := range a {
		sum[x.Key] += x.Count
	}
	for _, x := range b {
		sum[x.Key] += x.Count
	}
	out := make([]store.Bucket, 0, len(sum))
	for k, c := range sum {
		out = append(out, store.Bucket{Key: k, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

func mergePoints(a, b []store.Point) []store.Point {
	sum := make(map[int64]*store.Point, len(a)+len(b))
	add := func(ps []store.Point) {
		for _, p := range ps {
			k := p.At.UnixNano()
			if e, ok := sum[k]; ok {
				e.Count += p.Count
				continue
			}
			cp := p
			sum[k] = &cp
		}
	}
	add(a)
	add(b)
	out := make([]store.Point, 0, len(sum))
	for _, p := range sum {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

func mergeSeries(a, b []store.Series, n int) []store.Series {
	byKey := map[string][]store.Point{}
	var order []string
	add := func(ss []store.Series) {
		for _, s := range ss {
			if _, ok := byKey[s.Key]; !ok {
				order = append(order, s.Key)
			}
			byKey[s.Key] = mergePoints(byKey[s.Key], s.Points)
		}
	}
	add(a)
	add(b)

	type ranked struct {
		series store.Series
		total  int64
	}
	rs := make([]ranked, 0, len(order))
	for _, k := range order {
		var t int64
		for _, p := range byKey[k] {
			t += p.Count
		}
		rs = append(rs, ranked{store.Series{Key: k, Points: byKey[k]}, t})
	}
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].total != rs[j].total {
			return rs[i].total > rs[j].total
		}
		return rs[i].series.Key < rs[j].series.Key
	})
	if n > 0 && len(rs) > n {
		rs = rs[:n]
	}
	out := make([]store.Series, len(rs))
	for i, r := range rs {
		out[i] = r.series
	}
	return out
}

// mergeGroups combines two group lists level by level. Hits are the primary's
// first, then the legacy's: both are newest-first and the primary is newer.
func mergeGroups(a, b []store.Group, opts store.GroupOpts) []store.Group {
	idx := map[string]int{}
	var out []store.Group

	add := func(gs []store.Group) {
		for _, g := range gs {
			i, ok := idx[g.Key]
			if !ok {
				idx[g.Key] = len(out)
				out = append(out, g)
				continue
			}
			e := &out[i]
			e.Count += g.Count
			e.Children = mergeGroups(e.Children, g.Children, store.GroupOpts{Limit: opts.Limit, SortBy: opts.SortBy, Order: opts.Order, TopHits: opts.TopHits})
			e.Hits = append(e.Hits, g.Hits...)
		}
	}
	add(a)
	add(b)

	for i := range out {
		if opts.TopHits > 0 && len(out[i].Hits) > opts.TopHits {
			out[i].Hits = out[i].Hits[:opts.TopHits]
		}
	}

	byKey := strings.EqualFold(opts.SortBy, "key") || strings.EqualFold(opts.SortBy, "_key")
	asc := opts.Order == store.Asc
	sort.SliceStable(out, func(i, j int) bool {
		if byKey {
			if asc {
				return out[i].Key < out[j].Key
			}
			return out[i].Key > out[j].Key
		}
		if out[i].Count != out[j].Count {
			if asc {
				return out[i].Count < out[j].Count
			}
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out
}

func mergeFields(prim, leg []store.Field) []store.Field {
	byName := make(map[string]store.Field, len(prim)+len(leg))
	order := make([]string, 0, len(prim)+len(leg))
	for _, f := range prim {
		byName[f.Name] = f
		order = append(order, f.Name)
	}
	for _, f := range leg {
		p, ok := byName[f.Name]
		if !ok {
			byName[f.Name] = f
			order = append(order, f.Name)
			continue
		}
		// A field both stores know is only as capable as the weaker of them,
		// because a query on it reaches both.
		p.Filterable = p.Filterable && f.Filterable
		p.Searchable = p.Searchable && f.Searchable
		p.Conflict = p.Conflict || f.Conflict || p.Type != f.Type
		byName[f.Name] = p
	}
	out := make([]store.Field, len(order))
	for i, n := range order {
		out[i] = byName[n]
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// mergeSorted merges two lists already sorted by field into one page's worth.
// Records missing the field sort last whichever way the order runs.
func mergeSorted(a, b []json.RawMessage, field string, order store.SortOrder) []json.RawMessage {
	all := make([]json.RawMessage, 0, len(a)+len(b))
	all = append(all, a...)
	all = append(all, b...)

	keys := make([]gjson.Result, len(all))
	for i, d := range all {
		keys[i] = gjson.GetBytes(d, field)
	}
	idx := make([]int, len(all))
	for i := range idx {
		idx[i] = i
	}
	asc := order == store.Asc
	sort.SliceStable(idx, func(x, y int) bool {
		kx, ky := keys[idx[x]], keys[idx[y]]
		switch {
		case !kx.Exists() && !ky.Exists():
			return false
		case !kx.Exists():
			return false
		case !ky.Exists():
			return true
		}
		c := compareResults(kx, ky)
		if asc {
			return c < 0
		}
		return c > 0
	})

	out := make([]json.RawMessage, len(all))
	for i, j := range idx {
		out[i] = all[j]
	}
	return out
}

func compareResults(a, b gjson.Result) int {
	if a.Type == gjson.Number && b.Type == gjson.Number {
		switch {
		case a.Num < b.Num:
			return -1
		case a.Num > b.Num:
			return 1
		}
		return 0
	}
	return strings.Compare(a.String(), b.String())
}
