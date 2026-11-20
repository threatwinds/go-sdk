// Package factory builds a store from configuration, so that every process of
// an installation — the event processor, the backend, the plugins — reaches
// its data the same way and the choice of engine is a setting, not code.
//
// The configuration names a primary store and, optionally, a secondary one that
// is read-only and holds what came before a cutover. With a secondary configured
// the result is a composite store; without one it is the primary alone.
//
//	primary:
//	  engine: clickhouse
//	  clickhouse:
//	    addr: ["clickhouse:9000"]
//	    database: utmstack
//	    tenantColumn: tenantId
//	    tables: {logs: logs, alerts: alerts}
//	secondary:
//	  engine: opensearch
//	  opensearch:
//	    urls: ["https://opensearch:9200"]
//	    tenantField: tenantId
//	    layout: {logs: {prefix: v11-log, includeDataType: true, dateFormat: "2006.01.02"}}
//	  cutover: "2026-11-01T00:00:00Z"
//	  expires: "2026-12-01T00:00:00Z"
//	  tenant: ce66672c-e36d-4761-a8c8-90058fee1a24
//	  datasets: [logs]
//	  fieldPrefix: {primary: log, secondary: event}
//	  disabled: false
//
// Switching the secondary off is a setting too: secondary.disabled set to true keeps
// the rest of the section where it is. A process that starts with it set never
// connects to the secondary and answers from the primary alone. The configuration
// is read when a process starts, so one that was already running has to be
// restarted to see it.
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
	Primary   Engine     `json:"primary"`
	Secondary *Secondary `json:"secondary,omitempty"`
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

// Secondary is a read-only store holding what came before Cutover.
type Secondary struct {
	Engine

	// Cutover is when the secondary stopped receiving writes (RFC 3339).
	Cutover string `json:"cutover"`
	// Expires is when the secondary is switched off (RFC 3339); empty means never.
	Expires string `json:"expires,omitempty"`
	// Disabled switches the secondary off without removing its settings: nothing
	// connects to it and the rest of the section is not even checked, since it
	// is no longer used. The index pattern resolver still works, because v11
	// rules keep naming the indices they were written against.
	Disabled bool `json:"disabled,omitempty"`
	// Tenant is the one tenant the secondary data belongs to.
	Tenant string `json:"tenant"`
	// Datasets lists the datasets the secondary serves.
	Datasets []string `json:"datasets"`
	// TolerateFailure answers from the primary when the secondary fails, instead
	// of failing the read.
	TolerateFailure bool `json:"tolerateFailure,omitempty"`
	// FieldPrefix maps the object that holds parsed fields: the primary's name
	// for it and the secondary's ("log" and "event"). It applies to field names in
	// queries and to the records that come back.
	FieldPrefix *FieldPrefix `json:"fieldPrefix,omitempty"`
}

type FieldPrefix struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

// Options are what cannot come from a file.
type Options struct {
	// OnSecondaryError is told about a secondary failure that TolerateFailure hid.
	OnSecondaryError func(op string, err error)
	// OnReject is told about every row a ClickHouse writer permanently drops.
	OnReject func(row []byte, err error)
	// Now is the clock, replaceable in tests.
	Now func() time.Time
}

// Built is what New makes.
type Built struct {
	// Store is the store to use.
	Store store.Store

	// Composite is Store when a secondary is configured and enabled, and nil
	// otherwise. It reports whether the secondary is still reachable.
	Composite *composite.Store

	// SecondaryDisabled is true when a secondary is configured but switched off.
	SecondaryDisabled bool

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

// connectOpenSearch is how an OpenSearch store is connected; a test replaces it
// to see whether anything tried.
var connectOpenSearch = sdkos.Connect

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

	if cfg.Secondary == nil {
		b.Store = primary
		b.ResolveIndexPattern = resolverFor(cfg.Primary)
		return b, nil
	}

	if cfg.Secondary.Disabled {
		b.Store = primary
		b.SecondaryDisabled = true
		// v11 rules still name the indices they were written against, so the
		// layout that reads those names stays available.
		if r := resolverFor(cfg.Secondary.Engine); r != nil {
			b.ResolveIndexPattern = r
		} else {
			b.ResolveIndexPattern = resolverFor(cfg.Primary)
		}
		return b, nil
	}

	secondary, err := b.engine(cfg.Secondary.Engine, "secondary", opts)
	if err != nil {
		return fail(err)
	}

	cutover, _ := time.Parse(time.RFC3339, cfg.Secondary.Cutover)
	var expires time.Time
	if cfg.Secondary.Expires != "" {
		expires, _ = time.Parse(time.RFC3339, cfg.Secondary.Expires)
	}

	datasets := make([]store.Dataset, len(cfg.Secondary.Datasets))
	for i, d := range cfg.Secondary.Datasets {
		datasets[i] = store.Dataset(d)
	}

	cc := composite.Config{
		Primary:           primary,
		Secondary:         secondary,
		Cutover:           cutover,
		SecondaryExpires:  expires,
		SecondaryTenant:   cfg.Secondary.Tenant,
		SecondaryDatasets: datasets,
		TolerateSecondary: cfg.Secondary.TolerateFailure,
		OnSecondaryError:  opts.OnSecondaryError,
		Now:               opts.Now,
	}
	if fp := cfg.Secondary.FieldPrefix; fp != nil {
		cc.SecondaryField, cc.SecondaryFieldBack = composite.PrefixMap(fp.Primary, fp.Secondary)
		cc.SecondaryDocument = composite.RenameDocKey(fp.Secondary, fp.Primary)
	}

	comp, err := composite.New(cc)
	if err != nil {
		return fail(err)
	}
	b.Store = comp
	b.Composite = comp

	// A v11 rule names secondary indices, so the secondary's layout is the one that
	// reads them; the primary's is the fallback when the secondary is not OpenSearch.
	if r := resolverFor(cfg.Secondary.Engine); r != nil {
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
	l := cfg.Secondary
	if l == nil || l.Disabled {
		return nil
	}
	if err := validateEngine(l.Engine, "secondary"); err != nil {
		return err
	}

	cutover, err := time.Parse(time.RFC3339, l.Cutover)
	if err != nil {
		return fmt.Errorf("store config: secondary.cutover must be an RFC 3339 time: %w", err)
	}
	if l.Expires != "" {
		expires, err := time.Parse(time.RFC3339, l.Expires)
		if err != nil {
			return fmt.Errorf("store config: secondary.expires must be an RFC 3339 time: %w", err)
		}
		if !expires.After(cutover) {
			return errors.New("store config: secondary.expires must be after secondary.cutover")
		}
	}
	if fp := l.FieldPrefix; fp != nil && (fp.Primary == "" || fp.Secondary == "" || fp.Primary == fp.Secondary) {
		return errors.New("store config: secondary.fieldPrefix needs a primary and a secondary name that differ")
	}
	if l.Tenant == "" {
		return errors.New("store config: secondary.tenant is required, so that no other tenant can read the secondary data")
	}
	if len(l.Datasets) == 0 {
		return errors.New("store config: secondary.datasets must list the datasets the secondary serves")
	}

	// The OpenSearch client is one per process, so two OpenSearch stores can
	// only be the same cluster.
	p, lo := cfg.Primary.OpenSearch, l.OpenSearch
	if cfg.Primary.Engine == EngineOpenSearch && l.Engine.Engine == EngineOpenSearch && p != nil && lo != nil {
		if !slices.Equal(p.URLs, lo.URLs) || p.Username != lo.Username || p.Password != lo.Password {
			return errors.New("store config: primary and secondary are both OpenSearch but name different clusters; the client is shared by the process, so they must be the same one")
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
		if err := connectOpenSearch(o.URLs, o.Username, o.Password); err != nil {
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
