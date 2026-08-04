package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
)

const (
	defaultTimestampField = "@timestamp"
	defaultBucketLimit    = 10
)

type Config struct {
	Layout Layout

	// TenantField is required: a default would let a typo silently disable
	// tenant isolation.
	TenantField string

	TimestampField    string // defaults to @timestamp
	RetentionPolicyID string // ISM policy; empty disables Retention
}

// Driver holds no connection: the os package keeps a process-wide client.
type Driver struct {
	cfg    Config
	mapper *sdkos.FieldMapper
	now    func() time.Time
}

func New(cfg Config) (*Driver, error) {
	if cfg.Layout == nil {
		return nil, fmt.Errorf("opensearch: Config.Layout is required")
	}
	if cfg.TenantField == "" {
		return nil, fmt.Errorf("opensearch: Config.TenantField is required")
	}
	if cfg.TimestampField == "" {
		cfg.TimestampField = defaultTimestampField
	}
	return &Driver{cfg: cfg, mapper: sdkos.NewFieldMapper(), now: time.Now}, nil
}

var _ store.Store = (*Driver)(nil)

func (d *Driver) Count(ctx context.Context, s store.Scope, filters []store.Filter) (int64, error) {
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return 0, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{"size": 0, "query": q})
	if err != nil {
		return 0, err
	}
	return res.Hits.Total.Value, nil
}

func (d *Driver) CountInWindow(ctx context.Context, s store.Scope, filters []store.Filter, window time.Duration) (int64, error) {
	if window <= 0 {
		return 0, fmt.Errorf("opensearch: CountInWindow needs a positive window")
	}
	end := s.To
	if end.IsZero() {
		end = d.now()
	}
	s.From, s.To = end.Add(-window), end
	return d.Count(ctx, s, filters)
}

func (d *Driver) FetchByID(ctx context.Context, s store.Scope, id string) (json.RawMessage, error) {
	if id == "" {
		return nil, fmt.Errorf("opensearch: FetchByID needs an id")
	}
	idx, q, err := d.prepare(s, nil)
	if err != nil {
		return nil, err
	}
	if err := appendFilter(q, map[string]any{"ids": map[string]any{"values": []string{id}}}); err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{"size": 1, "query": q})
	if err != nil {
		return nil, err
	}
	if len(res.Hits.Hits) == 0 {
		return nil, store.ErrNotFound
	}
	return json.Marshal(res.Hits.Hits[0].Source)
}

func (d *Driver) FetchN(ctx context.Context, s store.Scope, filters []store.Filter, n int) ([]json.RawMessage, error) {
	if n <= 0 {
		return nil, nil
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{"size": n, "query": q})
	if err != nil {
		return nil, err
	}
	return sources(res.Hits.Hits)
}

func (d *Driver) FetchPage(ctx context.Context, s store.Scope, filters []store.Filter, page store.Page) ([]json.RawMessage, int64, error) {
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, 0, err
	}
	body := map[string]any{
		"from":             page.Offset,
		"size":             page.Limit,
		"query":            q,
		"track_total_hits": true,
	}
	if page.SortBy != "" {
		order := page.Order
		if order == "" {
			order = store.Desc
		}
		body["sort"] = []map[string]any{{page.SortBy: map[string]any{"order": string(order)}}}
	}
	res, err := sdkos.RawSearch(ctx, idx, body)
	if err != nil {
		return nil, 0, err
	}
	docs, err := sources(res.Hits.Hits)
	if err != nil {
		return nil, 0, err
	}
	return docs, res.Hits.Total.Value, nil
}

func (d *Driver) TopValues(ctx context.Context, s store.Scope, field string, filters []store.Filter, n int) ([]store.Bucket, error) {
	if field == "" {
		return nil, fmt.Errorf("opensearch: TopValues needs a field")
	}
	if n <= 0 {
		n = defaultBucketLimit
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{
		"size":  0,
		"query": q,
		"aggs": map[string]any{
			"vals": map[string]any{"terms": map[string]any{"field": field, "size": n}},
		},
	})
	if err != nil {
		return nil, err
	}
	raw := aggBuckets(res.Aggregations, "vals")
	out := make([]store.Bucket, 0, len(raw))
	for _, b := range raw {
		out = append(out, store.Bucket{Key: bucketKey(b), Count: bucketCount(b)})
	}
	return out, nil
}

func (d *Driver) Timeline(ctx context.Context, s store.Scope, filters []store.Filter, iv store.Interval) ([]store.Point, error) {
	hist, err := d.dateHistogram(iv)
	if err != nil {
		return nil, err
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{
		"size":  0,
		"query": q,
		"aggs":  map[string]any{"tl": map[string]any{"date_histogram": hist}},
	})
	if err != nil {
		return nil, err
	}
	return points(aggBuckets(res.Aggregations, "tl")), nil
}

func (d *Driver) TimelineByField(ctx context.Context, s store.Scope, field string, filters []store.Filter, iv store.Interval, n int) ([]store.Series, error) {
	if field == "" {
		return nil, fmt.Errorf("opensearch: TimelineByField needs a field")
	}
	if n <= 0 {
		n = defaultBucketLimit
	}
	hist, err := d.dateHistogram(iv)
	if err != nil {
		return nil, err
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{
		"size":  0,
		"query": q,
		"aggs": map[string]any{
			"by": map[string]any{
				"terms": map[string]any{"field": field, "size": n},
				"aggs":  map[string]any{"tl": map[string]any{"date_histogram": hist}},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	outer := aggBuckets(res.Aggregations, "by")
	out := make([]store.Series, 0, len(outer))
	for _, ob := range outer {
		out = append(out, store.Series{Key: bucketKey(ob), Points: points(aggBuckets(ob, "tl"))})
	}
	return out, nil
}

func (d *Driver) GroupBy(ctx context.Context, s store.Scope, fields []string, filters []store.Filter, opts store.GroupOpts) ([]store.Group, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("opensearch: GroupBy needs at least one field")
	}
	if opts.Limit <= 0 {
		opts.Limit = defaultBucketLimit
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return nil, err
	}
	res, err := sdkos.RawSearch(ctx, idx, map[string]any{
		"size":  0,
		"query": q,
		"aggs":  nestedTermsAgg(fields, 0, opts),
	})
	if err != nil {
		return nil, err
	}
	return collectGroups(res.Aggregations, fields, 0), nil
}

func (d *Driver) Insert(ctx context.Context, s store.Scope, id string, doc any) error {
	if s.Tenant == "" {
		return store.ErrNoTenant
	}
	idx, err := d.cfg.Layout.WriteIndex(s, d.now())
	if err != nil {
		return err
	}
	return sdkos.IndexDoc(ctx, doc, idx, id)
}

func (d *Driver) UpdateWhere(ctx context.Context, s store.Scope, filters []store.Filter, patch map[string]any) (int64, error) {
	if len(patch) == 0 {
		return 0, nil
	}
	idx, q, err := d.prepare(s, filters)
	if err != nil {
		return 0, err
	}

	var script strings.Builder
	params := make(map[string]any, len(patch))
	i := 0
	for field, val := range patch {
		key := "p" + strconv.Itoa(i)
		fmt.Fprintf(&script, "ctx._source.%s = params.%s; ", field, key)
		params[key] = val
		i++
	}

	raw, err := sdkos.UpdateByQuery(ctx, idx, map[string]any{
		"query": q,
		"script": map[string]any{
			"source": script.String(),
			"lang":   "painless",
			"params": params,
		},
	})
	if err != nil {
		return 0, err
	}
	var resp struct {
		Updated int64 `json:"updated"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return 0, fmt.Errorf("opensearch: decoding update_by_query response: %w", err)
	}
	return resp.Updated, nil
}

func (d *Driver) Usage(ctx context.Context) ([]store.DatasetUsage, error) {
	var out []store.DatasetUsage
	for _, ds := range d.cfg.Layout.Datasets() {
		patterns, err := d.cfg.Layout.ReadIndices(store.Scope{Dataset: ds})
		if err != nil {
			return nil, err
		}
		for _, p := range patterns {
			cats, err := sdkos.ListIndices(ctx, p)
			if err != nil {
				return nil, err
			}
			byType := map[string]*store.DatasetUsage{}
			for _, c := range cats {
				dt := d.cfg.Layout.DataTypeOf(ds, c.Index)
				u, ok := byType[dt]
				if !ok {
					u = &store.DatasetUsage{Dataset: ds, DataType: dt}
					byType[dt] = u
				}
				u.Documents += parseInt(c.DocsCount)
				u.Bytes += parseBytes(c.StoreSize)
				if t := parseEpochMillis(c.CreationDate); !t.IsZero() {
					if u.Oldest.IsZero() || t.Before(u.Oldest) {
						u.Oldest = t
					}
					if t.After(u.Newest) {
						u.Newest = t
					}
				}
			}
			for _, u := range byType {
				out = append(out, *u)
			}
		}
	}
	return out, nil
}

// Retention reads one ISM policy, so every dataset it covers reports the same.
func (d *Driver) Retention(ctx context.Context, dataset store.Dataset) (store.Retention, error) {
	policy, err := d.readPolicy(ctx)
	if err != nil {
		return store.Retention{}, err
	}
	age, coldAge := policyRetention(policy)
	keep, err := parseIndexAge(age)
	if err != nil {
		return store.Retention{}, err
	}

	var cold time.Duration
	if coldAge != "" {
		if cold, err = parseIndexAge(coldAge); err != nil {
			return store.Retention{}, err
		}
	}
	return store.Retention{Keep: keep, ColdAfter: cold}, nil
}

func (d *Driver) Health(ctx context.Context) (store.Health, error) {
	h, err := sdkos.GetClusterHealth(ctx)
	if err != nil {
		return store.Health{Status: store.HealthUnavailable, Message: err.Error()}, err
	}

	out := store.Health{Message: fmt.Sprintf("cluster %s, %d node(s)", h.ClusterName, h.NumberOfNodes)}
	switch h.Status {
	case "green":
		out.Status = store.HealthOK
	case "yellow":
		out.Status = store.HealthDegraded
	default:
		out.Status = store.HealthUnavailable
	}
	if pct, err := d.diskUsedPct(ctx); err == nil {
		out.DiskUsedPct = pct
	}
	return out, nil
}

// diskUsedPct reports the fullest node, which is what trips the watermarks.
func (d *Driver) diskUsedPct(ctx context.Context) (float64, error) {
	body, status, err := sdkos.DoRequest(ctx, "GET", "/_cat/allocation?format=json", nil)
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return 0, fmt.Errorf("opensearch: _cat/allocation returned %d", status)
	}
	var rows []struct {
		DiskPercent string `json:"disk.percent"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, err
	}
	var max float64
	for _, r := range rows {
		if v, err := strconv.ParseFloat(strings.TrimSpace(r.DiskPercent), 64); err == nil && v > max {
			max = v
		}
	}
	return max, nil
}

// prepare is the single entry point for reads, so tenant scoping cannot be missed.
func (d *Driver) prepare(s store.Scope, filters []store.Filter) ([]string, map[string]any, error) {
	idx, err := d.cfg.Layout.ReadIndices(s)
	if err != nil {
		return nil, nil, err
	}
	q, err := d.buildQuery(s, filters)
	if err != nil {
		return nil, nil, err
	}
	return idx, q, nil
}

func (d *Driver) dateHistogram(iv store.Interval) (map[string]any, error) {
	key, val, err := intervalSpec(iv)
	if err != nil {
		return nil, err
	}
	return map[string]any{"field": d.cfg.TimestampField, key: val, "min_doc_count": 0}, nil
}

func appendFilter(q map[string]any, clause map[string]any) error {
	b, ok := q["bool"].(map[string]any)
	if !ok {
		return fmt.Errorf("opensearch: query is not a bool query")
	}
	existing, ok := b["filter"].([]map[string]any)
	if !ok {
		return fmt.Errorf("opensearch: bool query has no filter clause")
	}
	b["filter"] = append(existing, clause)
	return nil
}

func nestedTermsAgg(fields []string, depth int, opts store.GroupOpts) map[string]any {
	terms := map[string]any{"field": fields[depth], "size": opts.Limit}
	if opts.SortBy != "" {
		order := opts.Order
		if order == "" {
			order = store.Desc
		}
		terms["order"] = map[string]any{opts.SortBy: string(order)}
	}

	level := map[string]any{"terms": terms}
	switch {
	case depth+1 < len(fields):
		level["aggs"] = nestedTermsAgg(fields, depth+1, opts)
	case opts.TopHits > 0:
		level["aggs"] = map[string]any{
			"hits": map[string]any{"top_hits": map[string]any{"size": opts.TopHits}},
		}
	}
	return map[string]any{groupAggName(depth): level}
}

func groupAggName(depth int) string { return "g" + strconv.Itoa(depth) }

func collectGroups(aggs map[string]any, fields []string, depth int) []store.Group {
	raw := aggBuckets(aggs, groupAggName(depth))
	out := make([]store.Group, 0, len(raw))
	for _, b := range raw {
		g := store.Group{Field: fields[depth], Key: bucketKey(b), Count: bucketCount(b)}
		if depth+1 < len(fields) {
			g.Children = collectGroups(b, fields, depth+1)
		}
		g.Hits = topHits(b)
		out = append(out, g)
	}
	return out
}

func topHits(b map[string]any) []json.RawMessage {
	node, ok := b["hits"].(map[string]any)
	if !ok {
		return nil
	}
	inner, ok := node["hits"].(map[string]any)
	if !ok {
		return nil
	}
	list, ok := inner["hits"].([]any)
	if !ok {
		return nil
	}
	out := make([]json.RawMessage, 0, len(list))
	for _, h := range list {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		raw, err := json.Marshal(hm["_source"])
		if err != nil {
			continue
		}
		out = append(out, raw)
	}
	return out
}

func points(buckets []map[string]any) []store.Point {
	out := make([]store.Point, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, store.Point{At: bucketTime(b), Count: bucketCount(b)})
	}
	return out
}

func sources(hits []sdkos.Hit) ([]json.RawMessage, error) {
	out := make([]json.RawMessage, 0, len(hits))
	for _, h := range hits {
		raw, err := json.Marshal(h.Source)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}

func parseInt(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func parseEpochMillis(s string) time.Time {
	ms, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func parseBytes(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	units := []struct {
		suffix string
		mult   float64
	}{
		{"tb", 1 << 40}, {"gb", 1 << 30}, {"mb", 1 << 20}, {"kb", 1 << 10}, {"b", 1},
	}
	for _, u := range units {
		if !strings.HasSuffix(s, u.suffix) {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, u.suffix), 64)
		if err != nil {
			return 0
		}
		return int64(v * u.mult)
	}
	return 0
}
