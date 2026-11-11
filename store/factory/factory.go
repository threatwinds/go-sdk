// Package factory builds a store from configuration, so that every process of
// an installation — the event processor, the backend, the plugins — reaches
// its data the same way and the choice of engine is a setting, not code.
//
// The configuration names a primary store and, optionally, a legacy one that
// is read-only and holds what came before a cutover. With a legacy configured
// the result is a composite store; without one it is the primary alone.
//
//	primary:
//	  engine: clickhouse
//	  clickhouse:
//	    addr: ["clickhouse:9000"]
//	    database: utmstack
//	    tenantColumn: tenantId
//	    tables: {logs: logs, alerts: alerts}
//	legacy:
//	  engine: opensearch
//	  opensearch:
//	    urls: ["https://opensearch:9200"]
//	    tenantField: tenantId
//	    layout: {logs: {prefix: v11-log, includeDataType: true, dateFormat: "2006.01.02"}}
//	  cutover: "2026-11-01T00:00:00Z"
//	  expires: "2026-12-01T00:00:00Z"
//	  tenant: ce66672c-e36d-4761-a8c8-90058fee1a24
//	  datasets: [logs]
//	  fieldPrefix: {current: log, legacy: event}
package factory

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"sigs.k8s.io/yaml"

	sdkos "github.com/threatwinds/go-sdk/os"
	"github.com/threatwinds/go-sdk/store"
	"github.com/threatwinds/go-sdk/store/clickhouse"
	"github.com/threatwinds/go-sdk/store/composite"
	"github.com/threatwinds/go-sdk/store/opensearch"
)

const (
	EngineClickHouse = "clickhouse"
	EngineOpenSearch = "opensearch"
)

type Config struct {
	Primary Engine  `json:"primary"`
	Legacy  *Legacy `json:"legacy,omitempty"`
}

// Engine selects one engine and carries the settings of the one it selects.
type Engine struct {
	Engine     string      `json:"engine"`
	ClickHouse *ClickHouse `json:"clickhouse,omitempty"`
	OpenSearch *OpenSearch `json:"opensearch,omitempty"`
}

type ClickHouse struct {
	Addr           []string          `json:"addr"`
	Database       string            `json:"database"`
	Username       string            `json:"username"`
	Password       string            `json:"password"`
	Tables         map[string]string `json:"tables"`
	TenantColumn   string            `json:"tenantColumn"`
	TimeColumn     string            `json:"timeColumn,omitempty"`
	DataTypeColumn string            `json:"dataTypeColumn,omitempty"`
	TextColumns    map[string]string `json:"textColumns,omitempty"`
	IDColumns      map[string]string `json:"idColumns,omitempty"`
	ColdVolume     string            `json:"coldVolume,omitempty"`
}

type OpenSearch struct {
	URLs              []string                 `json:"urls"`
	Username          string                   `json:"username"`
	Password          string                   `json:"password"`
	TenantField       string                   `json:"tenantField"`
	TimestampField    string                   `json:"timestampField,omitempty"`
	RetentionPolicyID string                   `json:"retentionPolicyId,omitempty"`
	Layout            map[string]DatasetLayout `json:"layout"`
	TextFields        map[string]string        `json:"textFields,omitempty"`
}

// DatasetLayout is opensearch.DatasetSpec: how a dataset's indices are named.
type DatasetLayout struct {
	Prefix          string `json:"prefix"`
	IncludeDataType bool   `json:"includeDataType,omitempty"`
	DateFormat      string `json:"dateFormat,omitempty"`
}

// Legacy is a read-only store holding what came before Cutover.
type Legacy struct {
	Engine

	// Cutover is when the legacy stopped receiving writes (RFC 3339).
	Cutover string `json:"cutover"`
	// Expires is when the legacy is switched off (RFC 3339); empty means never.
	Expires string `json:"expires,omitempty"`
	// Tenant is the one tenant the legacy data belongs to.
	Tenant string `json:"tenant"`
	// Datasets lists the datasets the legacy serves.
	Datasets []string `json:"datasets"`
	// TolerateFailure answers from the primary when the legacy fails, instead
	// of failing the read.
	TolerateFailure bool `json:"tolerateFailure,omitempty"`
	// FieldPrefix maps the object that holds parsed fields: the primary's name
	// for it and the legacy's ("log" and "event"). It applies to field names in
	// queries and to the records that come back.
	FieldPrefix *FieldPrefix `json:"fieldPrefix,omitempty"`
}

type FieldPrefix struct {
	Current string `json:"current"`
	Legacy  string `json:"legacy"`
}

// Options are what cannot come from a file.
type Options struct {
	// OnLegacyError is told about a legacy failure that TolerateFailure hid.
	OnLegacyError func(op string, err error)
	// OnReject is told about every row a ClickHouse writer permanently drops.
	OnReject func(row []byte, err error)
	// Now is the clock, replaceable in tests.
	Now func() time.Time
}

// Built is what New makes.
type Built struct {
	// Store is the store to use.
	Store store.Store

	// Composite is Store when a legacy is configured, and nil otherwise. It
	// reports whether the legacy is still reachable.
	Composite *composite.Store

	// ResolveIndexPattern turns the index pattern a v11 rule names into a
	// dataType. It is nil when no OpenSearch layout is configured. Install it
	// with plugins.SetIndexPatternResolver.
	ResolveIndexPattern func(dataset store.Dataset, pattern string) (dataType string, ok bool)

	closers []func() error
}

// Close releases what New opened.
func (b *Built) Close() error {
	var errs []error
	for _, c := range b.closers {
		if err := c(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Parse reads YAML or JSON. An unknown field is an error, so a misspelt setting
// is found at startup rather than ignored.
func Parse(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.UnmarshalStrict(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("store config: %w", err)
	}
	return cfg, nil
}

// New builds the store the configuration describes.
func New(cfg Config, opts Options) (*Built, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	b := &Built{}
	fail := func(err error) (*Built, error) {
		_ = b.Close()
		return nil, err
	}

	primary, err := b.engine(cfg.Primary, "primary", opts)
	if err != nil {
		return fail(err)
	}

	if cfg.Legacy == nil {
		b.Store = primary
		b.ResolveIndexPattern = resolverFor(cfg.Primary)
		return b, nil
	}

	legacy, err := b.engine(cfg.Legacy.Engine, "legacy", opts)
	if err != nil {
		return fail(err)
	}

	cutover, _ := time.Parse(time.RFC3339, cfg.Legacy.Cutover)
	var expires time.Time
	if cfg.Legacy.Expires != "" {
		expires, _ = time.Parse(time.RFC3339, cfg.Legacy.Expires)
	}

	datasets := make([]store.Dataset, len(cfg.Legacy.Datasets))
	for i, d := range cfg.Legacy.Datasets {
		datasets[i] = store.Dataset(d)
	}

	cc := composite.Config{
		Primary:        primary,
		Legacy:         legacy,
		Cutover:        cutover,
		LegacyExpires:  expires,
		LegacyTenant:   cfg.Legacy.Tenant,
		LegacyDatasets: datasets,
		TolerateLegacy: cfg.Legacy.TolerateFailure,
		OnLegacyError:  opts.OnLegacyError,
		Now:            opts.Now,
	}
	if fp := cfg.Legacy.FieldPrefix; fp != nil {
		cc.LegacyField, cc.LegacyFieldBack = composite.PrefixMap(fp.Current, fp.Legacy)
		cc.LegacyDocument = composite.RenameDocKey(fp.Legacy, fp.Current)
	}

	comp, err := composite.New(cc)
	if err != nil {
		return fail(err)
	}
	b.Store = comp
	b.Composite = comp

	// A v11 rule names legacy indices, so the legacy's layout is the one that
	// reads them; the primary's is the fallback when the legacy is not OpenSearch.
	if r := resolverFor(cfg.Legacy.Engine); r != nil {
		b.ResolveIndexPattern = r
	} else {
		b.ResolveIndexPattern = resolverFor(cfg.Primary)
	}
	return b, nil
}

func validate(cfg Config) error {
	if err := validateEngine(cfg.Primary, "primary"); err != nil {
		return err
	}
	l := cfg.Legacy
	if l == nil {
		return nil
	}
	if err := validateEngine(l.Engine, "legacy"); err != nil {
		return err
	}

	cutover, err := time.Parse(time.RFC3339, l.Cutover)
	if err != nil {
		return fmt.Errorf("store config: legacy.cutover must be an RFC 3339 time: %w", err)
	}
	if l.Expires != "" {
		expires, err := time.Parse(time.RFC3339, l.Expires)
		if err != nil {
			return fmt.Errorf("store config: legacy.expires must be an RFC 3339 time: %w", err)
		}
		if !expires.After(cutover) {
			return errors.New("store config: legacy.expires must be after legacy.cutover")
		}
	}
	if fp := l.FieldPrefix; fp != nil && (fp.Current == "" || fp.Legacy == "" || fp.Current == fp.Legacy) {
		return errors.New("store config: legacy.fieldPrefix needs a current and a legacy name that differ")
	}
	if l.Tenant == "" {
		return errors.New("store config: legacy.tenant is required, so that no other tenant can read the legacy data")
	}
	if len(l.Datasets) == 0 {
		return errors.New("store config: legacy.datasets must list the datasets the legacy serves")
	}

	// The OpenSearch client is one per process, so two OpenSearch stores can
	// only be the same cluster.
	p, lo := cfg.Primary.OpenSearch, l.OpenSearch
	if cfg.Primary.Engine == EngineOpenSearch && l.Engine.Engine == EngineOpenSearch && p != nil && lo != nil {
		if !slices.Equal(p.URLs, lo.URLs) || p.Username != lo.Username || p.Password != lo.Password {
			return errors.New("store config: primary and legacy are both OpenSearch but name different clusters; the client is shared by the process, so they must be the same one")
		}
	}
	return nil
}

func validateEngine(e Engine, which string) error {
	switch e.Engine {
	case EngineClickHouse:
		if e.ClickHouse == nil {
			return fmt.Errorf("store config: %s.engine is clickhouse but %s.clickhouse is missing", which, which)
		}
	case EngineOpenSearch:
		if e.OpenSearch == nil {
			return fmt.Errorf("store config: %s.engine is opensearch but %s.opensearch is missing", which, which)
		}
		if len(e.OpenSearch.URLs) == 0 {
			return fmt.Errorf("store config: %s.opensearch.urls is required", which)
		}
	case "":
		return fmt.Errorf("store config: %s.engine is required (%s or %s)", which, EngineClickHouse, EngineOpenSearch)
	default:
		return fmt.Errorf("store config: %s.engine %q is not one of %s, %s", which, e.Engine, EngineClickHouse, EngineOpenSearch)
	}
	return nil
}

// engine opens one engine.
func (b *Built) engine(e Engine, which string, opts Options) (store.Store, error) {
	switch e.Engine {
	case EngineClickHouse:
		c := e.ClickHouse
		d, err := clickhouse.New(clickhouse.Config{
			Addr:           c.Addr,
			Database:       c.Database,
			Username:       c.Username,
			Password:       c.Password,
			Tables:         datasetMap(c.Tables),
			TenantColumn:   c.TenantColumn,
			TimeColumn:     c.TimeColumn,
			DataTypeColumn: c.DataTypeColumn,
			TextColumns:    datasetMap(c.TextColumns),
			IDColumns:      datasetMap(c.IDColumns),
			ColdVolume:     c.ColdVolume,
			OnReject:       opts.OnReject,
		})
		if err != nil {
			return nil, fmt.Errorf("store config: %s: %w", which, err)
		}
		b.closers = append(b.closers, d.Close)
		return d, nil

	default:
		o := e.OpenSearch
		if err := sdkos.Connect(o.URLs, o.Username, o.Password); err != nil {
			return nil, fmt.Errorf("store config: %s: connect to opensearch: %w", which, err)
		}
		d, err := opensearch.New(opensearch.Config{
			Layout:            layoutOf(o),
			TenantField:       o.TenantField,
			TimestampField:    o.TimestampField,
			RetentionPolicyID: o.RetentionPolicyID,
			TextFields:        datasetMap(o.TextFields),
		})
		if err != nil {
			return nil, fmt.Errorf("store config: %s: %w", which, err)
		}
		return d, nil
	}
}

func layoutOf(o *OpenSearch) opensearch.SpecLayout {
	l := make(opensearch.SpecLayout, len(o.Layout))
	for ds, spec := range o.Layout {
		l[store.Dataset(ds)] = opensearch.DatasetSpec{
			Prefix:          spec.Prefix,
			IncludeDataType: spec.IncludeDataType,
			DateFormat:      spec.DateFormat,
		}
	}
	return l
}

func resolverFor(e Engine) func(store.Dataset, string) (string, bool) {
	if e.Engine != EngineOpenSearch || e.OpenSearch == nil {
		return nil
	}
	return layoutOf(e.OpenSearch).DataTypeOfPattern
}

func datasetMap(m map[string]string) map[store.Dataset]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[store.Dataset]string, len(m))
	for k, v := range m {
		out[store.Dataset(k)] = v
	}
	return out
}
