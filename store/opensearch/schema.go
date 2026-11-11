package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
)

func (d *Driver) DescribeFields(ctx context.Context, s store.Scope) ([]store.Field, error) {
	idx, err := d.cfg.Layout.ReadIndices(s)
	if err != nil {
		return nil, err
	}
	if len(idx) == 0 {
		return nil, nil
	}

	merged, err := d.mapper.GetMergedMapping(ctx, idx[0])
	if err != nil {
		return nil, err
	}

	out := make([]store.Field, 0, len(merged.Fields))
	for name, f := range merged.Fields {
		out = append(out, store.Field{
			Name:       name,
			Type:       f.Type,
			Filterable: f.AllowsTerm,
			Searchable: f.AllowsMatch,
			Conflict:   f.HasConflict,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (d *Driver) Flush(ctx context.Context, dataset store.Dataset) error {
	idx, err := d.cfg.Layout.ReadIndices(store.Scope{Dataset: dataset})
	if err != nil {
		return err
	}
	return sdkos.RefreshIndex(ctx, idx...)
}

// Drop deletes by query rather than dropping indices, so a scoped delete can
// never reach another tenant's records sharing the same index.
func (d *Driver) Drop(ctx context.Context, s store.Scope) (int64, error) {
	idx, q, err := d.prepare(s, nil)
	if err != nil {
		return 0, err
	}

	path := "/" + joinIndices(idx) + "/_delete_by_query?refresh=true&conflicts=proceed"
	body, status, err := sdkos.DoRequest(ctx, "POST", path, map[string]any{"query": q})
	if err != nil {
		return 0, err
	}
	if status < 200 || status >= 300 {
		return 0, fmt.Errorf("opensearch: _delete_by_query returned %d: %s", status, string(body))
	}

	var resp struct {
		Deleted int64 `json:"deleted"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("opensearch: decoding _delete_by_query response: %w", err)
	}
	return resp.Deleted, nil
}

func joinIndices(idx []string) string {
	out := idx[0]
	for _, i := range idx[1:] {
		out += "," + i
	}
	return out
}
