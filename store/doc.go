// Package store is the interface every engine implements and the types its
// callers speak. The engines live in subpackages (clickhouse, opensearch), the
// composite that serves one logical store from two is in composite, and
// factory builds any of them from configuration.
//
// # Where the engines differ
//
// A filter is meant to mean the same on any engine, and almost all of them do.
// These do not, and a caller that relies on the difference has to know:
//
// Text matching. OpContains, OpNotContains, OpContainsAny and OpNotContainsAny
// are a case-insensitive substring on ClickHouse. On OpenSearch they are a
// "match" query, which compares analysed words: "deni" finds nothing in
// "denied" there, and word order and stemming play a part. OpSearch reads the
// raw-text column on ClickHouse and the field named in
// opensearch.Config.TextFields on OpenSearch, with every word required.
//
// Anchored matching. OpStartsWith and OpEndsWith are case-insensitive on both,
// but OpenSearch evaluates them against the indexed terms, so the field has to
// be a keyword. On an analysed text field they match individual words, not the
// whole value.
//
// Retention. ClickHouse can move data to a cold volume after
// Retention.ColdAfter. OpenSearch cannot: SetRetention changes Keep only, and
// asking it to add or remove a cold tier returns ErrUnsupported. Reading or
// setting retention there also needs an ISM policy id in the configuration.
//
// Raw queries. RawQuerier takes the engine's own query text, so what runs on
// one does not run on the other.
//
// Neither engine's behaviour has been changed to hide these: making
// OpenSearch answer a substring would need wildcard queries over keyword
// subfields, which depends on how an installation maps its indices.
package store
