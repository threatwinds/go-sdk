package os

import (
	"context"
	"fmt"
	"log"
)

// BoolBuilder builds bool queries with field resolution at all nesting levels
type BoolBuilder struct {
	ctx         context.Context
	mapper      *FieldMapper
	indices     []string
	query       Bool
	errors      []error
	processName string
}

// NewBoolBuilder creates a bool builder with field resolution using default mapper
func NewBoolBuilder(ctx context.Context, indices []string, processName string) *BoolBuilder {
	return NewBoolBuilderWithMapper(ctx, indices, defaultMapper, processName)
}

// NewBoolBuilderWithMapper creates a bool builder with custom mapper
func NewBoolBuilderWithMapper(ctx context.Context, indices []string, mapper *FieldMapper, processName string) *BoolBuilder {
	return &BoolBuilder{
		ctx:     ctx,
		mapper:  mapper,
		indices: indices,
		query: Bool{
			Must:    []Query{},
			Should:  []Query{},
			Filter:  []Query{},
			MustNot: []Query{},
		},
		errors:      []error{},
		processName: processName,
	}
}

// --- Term Methods ---

// MustTerm adds a term query to must clause
func (b *BoolBuilder) MustTerm(field string, value interface{}) *BoolBuilder {
	query, err := b.buildTermQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldTerm adds a term query to should clause
func (b *BoolBuilder) ShouldTerm(field string, value interface{}) *BoolBuilder {
	query, err := b.buildTermQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterTerm adds a term query to filter clause
func (b *BoolBuilder) FilterTerm(field string, value interface{}) *BoolBuilder {
	query, err := b.buildTermQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotTerm adds a term query to must_not clause
func (b *BoolBuilder) MustNotTerm(field string, value interface{}) *BoolBuilder {
	query, err := b.buildTermQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Terms Methods ---

// MustTerms adds a terms query to must clause
func (b *BoolBuilder) MustTerms(field string, values ...interface{}) *BoolBuilder {
	query, err := b.buildTermsQuery(field, values)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldTerms adds a terms query to should clause
func (b *BoolBuilder) ShouldTerms(field string, values ...interface{}) *BoolBuilder {
	query, err := b.buildTermsQuery(field, values)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterTerms adds a terms query to filter clause
func (b *BoolBuilder) FilterTerms(field string, values ...interface{}) *BoolBuilder {
	query, err := b.buildTermsQuery(field, values)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotTerms adds a terms query to must_not clause
func (b *BoolBuilder) MustNotTerms(field string, values ...interface{}) *BoolBuilder {
	query, err := b.buildTermsQuery(field, values)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- CIDR Methods (for "ip" type fields) ---
// These methods are for fields mapped as "ip" type (storing single IP addresses).
// For "ip_range" type fields, use the IPRangeContains/Intersects/Within methods below.

// MustCIDR adds a CIDR query to must clause for "ip" type fields
// Example: MustCIDR("client_ip", "192.168.0.0/16")
func (b *BoolBuilder) MustCIDR(field string, cidr string) *BoolBuilder {
	query := b.buildCIDRQuery(field, cidr)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldCIDR adds a CIDR query to should clause for "ip" type fields
func (b *BoolBuilder) ShouldCIDR(field string, cidr string) *BoolBuilder {
	query := b.buildCIDRQuery(field, cidr)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterCIDR adds a CIDR query to filter clause for "ip" type fields
func (b *BoolBuilder) FilterCIDR(field string, cidr string) *BoolBuilder {
	query := b.buildCIDRQuery(field, cidr)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotCIDR adds a CIDR query to must_not clause for "ip" type fields
func (b *BoolBuilder) MustNotCIDR(field string, cidr string) *BoolBuilder {
	query := b.buildCIDRQuery(field, cidr)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// MustCIDRs adds a terms query with multiple CIDRs to must clause for "ip" type fields
// Example: MustCIDRs("client_ip", "192.168.0.0/24", "10.0.0.0/8")
func (b *BoolBuilder) MustCIDRs(field string, cidrs ...string) *BoolBuilder {
	query := b.buildCIDRsQuery(field, cidrs)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldCIDRs adds a terms query with multiple CIDRs to should clause for "ip" type fields
func (b *BoolBuilder) ShouldCIDRs(field string, cidrs ...string) *BoolBuilder {
	query := b.buildCIDRsQuery(field, cidrs)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterCIDRs adds a terms query with multiple CIDRs to filter clause for "ip" type fields
func (b *BoolBuilder) FilterCIDRs(field string, cidrs ...string) *BoolBuilder {
	query := b.buildCIDRsQuery(field, cidrs)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotCIDRs adds a terms query with multiple CIDRs to must_not clause for "ip" type fields
func (b *BoolBuilder) MustNotCIDRs(field string, cidrs ...string) *BoolBuilder {
	query := b.buildCIDRsQuery(field, cidrs)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- IP Range Methods (for "ip_range" type fields) ---
// These methods are for fields mapped as "ip_range" type (storing IP address ranges).
// ip_range fields store ranges like {"gte": "10.0.0.0", "lte": "10.255.255.255"} or CIDR notation.

// MustIPRangeContains adds a query to must clause that matches ip_range fields containing the given IP
// Example: MustIPRangeContains("allowed_ips", "192.168.1.50") matches ranges like 192.168.0.0/16
func (b *BoolBuilder) MustIPRangeContains(field string, ip string) *BoolBuilder {
	query := b.buildIPRangeContainsQuery(field, ip)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldIPRangeContains adds a query to should clause that matches ip_range fields containing the given IP
func (b *BoolBuilder) ShouldIPRangeContains(field string, ip string) *BoolBuilder {
	query := b.buildIPRangeContainsQuery(field, ip)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterIPRangeContains adds a query to filter clause that matches ip_range fields containing the given IP
func (b *BoolBuilder) FilterIPRangeContains(field string, ip string) *BoolBuilder {
	query := b.buildIPRangeContainsQuery(field, ip)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotIPRangeContains adds a query to must_not clause that matches ip_range fields containing the given IP
func (b *BoolBuilder) MustNotIPRangeContains(field string, ip string) *BoolBuilder {
	query := b.buildIPRangeContainsQuery(field, ip)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// MustIPRangeIntersects adds a query to must clause that matches ip_range fields overlapping with the given range
// Example: MustIPRangeIntersects("blocked_ranges", "192.168.0.0", "192.168.255.255")
func (b *BoolBuilder) MustIPRangeIntersects(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeIntersectsQuery(field, fromIP, toIP)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldIPRangeIntersects adds a query to should clause that matches ip_range fields overlapping with the given range
func (b *BoolBuilder) ShouldIPRangeIntersects(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeIntersectsQuery(field, fromIP, toIP)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterIPRangeIntersects adds a query to filter clause that matches ip_range fields overlapping with the given range
func (b *BoolBuilder) FilterIPRangeIntersects(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeIntersectsQuery(field, fromIP, toIP)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotIPRangeIntersects adds a query to must_not clause that matches ip_range fields overlapping with the given range
func (b *BoolBuilder) MustNotIPRangeIntersects(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeIntersectsQuery(field, fromIP, toIP)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// MustIPRangeWithin adds a query to must clause that matches ip_range fields entirely within the given range
// Example: MustIPRangeWithin("subnet", "10.0.0.0", "10.255.255.255") matches 10.0.0.0/24, 10.1.0.0/16, etc.
func (b *BoolBuilder) MustIPRangeWithin(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeWithinQuery(field, fromIP, toIP)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldIPRangeWithin adds a query to should clause that matches ip_range fields entirely within the given range
func (b *BoolBuilder) ShouldIPRangeWithin(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeWithinQuery(field, fromIP, toIP)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterIPRangeWithin adds a query to filter clause that matches ip_range fields entirely within the given range
func (b *BoolBuilder) FilterIPRangeWithin(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeWithinQuery(field, fromIP, toIP)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotIPRangeWithin adds a query to must_not clause that matches ip_range fields entirely within the given range
func (b *BoolBuilder) MustNotIPRangeWithin(field string, fromIP, toIP string) *BoolBuilder {
	query := b.buildIPRangeWithinQuery(field, fromIP, toIP)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Match Methods ---

// MustMatch adds a match query to must clause
func (b *BoolBuilder) MustMatch(field string, value string) *BoolBuilder {
	query := b.buildMatchQuery(field, value)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldMatch adds a match query to should clause
func (b *BoolBuilder) ShouldMatch(field string, value string) *BoolBuilder {
	query := b.buildMatchQuery(field, value)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterMatch adds a match query to filter clause
func (b *BoolBuilder) FilterMatch(field string, value string) *BoolBuilder {
	query := b.buildMatchQuery(field, value)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotMatch adds a match query to must_not clause
func (b *BoolBuilder) MustNotMatch(field string, value string) *BoolBuilder {
	query := b.buildMatchQuery(field, value)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Range Methods ---

// MustRange adds a range query to must clause
func (b *BoolBuilder) MustRange(field string, op string, value interface{}) *BoolBuilder {
	query, err := b.buildRangeQuery(field, op, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldRange adds a range query to should clause
func (b *BoolBuilder) ShouldRange(field string, op string, value interface{}) *BoolBuilder {
	query, err := b.buildRangeQuery(field, op, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterRange adds a range query to filter clause
func (b *BoolBuilder) FilterRange(field string, op string, value interface{}) *BoolBuilder {
	query, err := b.buildRangeQuery(field, op, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotRange adds a range query to must_not clause
func (b *BoolBuilder) MustNotRange(field string, op string, value interface{}) *BoolBuilder {
	query, err := b.buildRangeQuery(field, op, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Exists Methods ---

// MustExists adds an exists query to must clause
func (b *BoolBuilder) MustExists(field string) *BoolBuilder {
	query := b.buildExistsQuery(field)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldExists adds an exists query to should clause
func (b *BoolBuilder) ShouldExists(field string) *BoolBuilder {
	query := b.buildExistsQuery(field)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterExists adds an exists query to filter clause
func (b *BoolBuilder) FilterExists(field string) *BoolBuilder {
	query := b.buildExistsQuery(field)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotExists adds an exists query to must_not clause
func (b *BoolBuilder) MustNotExists(field string) *BoolBuilder {
	query := b.buildExistsQuery(field)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Wildcard Methods ---

// MustWildcard adds a wildcard query to must clause
func (b *BoolBuilder) MustWildcard(field string, pattern string) *BoolBuilder {
	query := b.buildWildcardQuery(field, pattern)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldWildcard adds a wildcard query to should clause
func (b *BoolBuilder) ShouldWildcard(field string, pattern string) *BoolBuilder {
	query := b.buildWildcardQuery(field, pattern)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterWildcard adds a wildcard query to filter clause
func (b *BoolBuilder) FilterWildcard(field string, pattern string) *BoolBuilder {
	query := b.buildWildcardQuery(field, pattern)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotWildcard adds a wildcard query to must_not clause
func (b *BoolBuilder) MustNotWildcard(field string, pattern string) *BoolBuilder {
	query := b.buildWildcardQuery(field, pattern)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Nested Bool Methods ---

// MustBool adds a nested bool query to must clause
func (b *BoolBuilder) MustBool(nested *BoolBuilder) *BoolBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.query.Must = append(b.query.Must, Query{Bool: &nested.query})
	return b
}

// ShouldBool adds a nested bool query to should clause
func (b *BoolBuilder) ShouldBool(nested *BoolBuilder) *BoolBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.query.Should = append(b.query.Should, Query{Bool: &nested.query})
	return b
}

// FilterBool adds a nested bool query to filter clause
func (b *BoolBuilder) FilterBool(nested *BoolBuilder) *BoolBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.query.Filter = append(b.query.Filter, Query{Bool: &nested.query})
	return b
}

// MustNotBool adds a nested bool query to must_not clause
func (b *BoolBuilder) MustNotBool(nested *BoolBuilder) *BoolBuilder {
	if nested == nil {
		return b
	}
	b.errors = append(b.errors, nested.errors...)
	b.query.MustNot = append(b.query.MustNot, Query{Bool: &nested.query})
	return b
}

// --- Prefix Methods ---

// MustPrefix adds a prefix query to must clause
func (b *BoolBuilder) MustPrefix(field string, prefix string) *BoolBuilder {
	query, err := b.buildPrefixQuery(field, prefix)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldPrefix adds a prefix query to should clause
func (b *BoolBuilder) ShouldPrefix(field string, prefix string) *BoolBuilder {
	query, err := b.buildPrefixQuery(field, prefix)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterPrefix adds a prefix query to filter clause
func (b *BoolBuilder) FilterPrefix(field string, prefix string) *BoolBuilder {
	query, err := b.buildPrefixQuery(field, prefix)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotPrefix adds a prefix query to must_not clause
func (b *BoolBuilder) MustNotPrefix(field string, prefix string) *BoolBuilder {
	query, err := b.buildPrefixQuery(field, prefix)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- IDs Methods ---

// FilterIDs adds an IDs query to filter clause
func (b *BoolBuilder) FilterIDs(ids ...interface{}) *BoolBuilder {
	query := Query{
		IDs: map[string][]interface{}{"values": ids},
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotIDs adds an IDs query to must_not clause
func (b *BoolBuilder) MustNotIDs(ids ...interface{}) *BoolBuilder {
	query := Query{
		IDs: map[string][]interface{}{"values": ids},
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- QueryString Methods ---

// MustQueryString adds a query_string query to must clause
func (b *BoolBuilder) MustQueryString(query string, defaultOperator ...string) *BoolBuilder {
	qs := &QueryString{Query: query}
	if len(defaultOperator) > 0 {
		qs.DefaultOperator = defaultOperator[0]
	}
	b.query.Must = append(b.query.Must, Query{QueryString: qs})
	return b
}

// ShouldQueryString adds a query_string query to should clause
func (b *BoolBuilder) ShouldQueryString(query string, defaultOperator ...string) *BoolBuilder {
	qs := &QueryString{Query: query}
	if len(defaultOperator) > 0 {
		qs.DefaultOperator = defaultOperator[0]
	}
	b.query.Should = append(b.query.Should, Query{QueryString: qs})
	return b
}

// FilterQueryString adds a query_string query to filter clause
func (b *BoolBuilder) FilterQueryString(query string, defaultOperator ...string) *BoolBuilder {
	qs := &QueryString{Query: query}
	if len(defaultOperator) > 0 {
		qs.DefaultOperator = defaultOperator[0]
	}
	b.query.Filter = append(b.query.Filter, Query{QueryString: qs})
	return b
}

// MustNotQueryString adds a query_string query to must_not clause
func (b *BoolBuilder) MustNotQueryString(query string, defaultOperator ...string) *BoolBuilder {
	qs := &QueryString{Query: query}
	if len(defaultOperator) > 0 {
		qs.DefaultOperator = defaultOperator[0]
	}
	b.query.MustNot = append(b.query.MustNot, Query{QueryString: qs})
	return b
}

// --- MultiMatch Methods ---

// MustMultiMatch adds a multi_match query to must clause
func (b *BoolBuilder) MustMultiMatch(query string, fields []string, matchType ...string) *BoolBuilder {
	mm := &MultiMatch{Query: query, Fields: fields}
	if len(matchType) > 0 {
		mm.Type = matchType[0]
	}
	b.query.Must = append(b.query.Must, Query{MultiMatch: mm})
	return b
}

// ShouldMultiMatch adds a multi_match query to should clause
func (b *BoolBuilder) ShouldMultiMatch(query string, fields []string, matchType ...string) *BoolBuilder {
	mm := &MultiMatch{Query: query, Fields: fields}
	if len(matchType) > 0 {
		mm.Type = matchType[0]
	}
	b.query.Should = append(b.query.Should, Query{MultiMatch: mm})
	return b
}

// FilterMultiMatch adds a multi_match query to filter clause
func (b *BoolBuilder) FilterMultiMatch(query string, fields []string, matchType ...string) *BoolBuilder {
	mm := &MultiMatch{Query: query, Fields: fields}
	if len(matchType) > 0 {
		mm.Type = matchType[0]
	}
	b.query.Filter = append(b.query.Filter, Query{MultiMatch: mm})
	return b
}

// MustNotMultiMatch adds a multi_match query to must_not clause
func (b *BoolBuilder) MustNotMultiMatch(query string, fields []string, matchType ...string) *BoolBuilder {
	mm := &MultiMatch{Query: query, Fields: fields}
	if len(matchType) > 0 {
		mm.Type = matchType[0]
	}
	b.query.MustNot = append(b.query.MustNot, Query{MultiMatch: mm})
	return b
}

// --- Fuzzy Methods ---

// MustFuzzy adds a fuzzy query to must clause
func (b *BoolBuilder) MustFuzzy(field string, value string, fuzziness ...string) *BoolBuilder {
	query, err := b.buildFuzzyQuery(field, value, fuzziness...)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldFuzzy adds a fuzzy query to should clause
func (b *BoolBuilder) ShouldFuzzy(field string, value string, fuzziness ...string) *BoolBuilder {
	query, err := b.buildFuzzyQuery(field, value, fuzziness...)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterFuzzy adds a fuzzy query to filter clause
func (b *BoolBuilder) FilterFuzzy(field string, value string, fuzziness ...string) *BoolBuilder {
	query, err := b.buildFuzzyQuery(field, value, fuzziness...)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotFuzzy adds a fuzzy query to must_not clause
func (b *BoolBuilder) MustNotFuzzy(field string, value string, fuzziness ...string) *BoolBuilder {
	query, err := b.buildFuzzyQuery(field, value, fuzziness...)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- Regexp Methods ---

// MustRegexp adds a regexp query to must clause
func (b *BoolBuilder) MustRegexp(field string, pattern string) *BoolBuilder {
	query := b.buildRegexpQuery(field, pattern)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldRegexp adds a regexp query to should clause
func (b *BoolBuilder) ShouldRegexp(field string, pattern string) *BoolBuilder {
	query := b.buildRegexpQuery(field, pattern)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterRegexp adds a regexp query to filter clause
func (b *BoolBuilder) FilterRegexp(field string, pattern string) *BoolBuilder {
	query := b.buildRegexpQuery(field, pattern)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotRegexp adds a regexp query to must_not clause
func (b *BoolBuilder) MustNotRegexp(field string, pattern string) *BoolBuilder {
	query := b.buildRegexpQuery(field, pattern)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- MatchPhrasePrefix Methods ---

// MustMatchPhrasePrefix adds a match_phrase_prefix query to must clause
func (b *BoolBuilder) MustMatchPhrasePrefix(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhrasePrefixQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldMatchPhrasePrefix adds a match_phrase_prefix query to should clause
func (b *BoolBuilder) ShouldMatchPhrasePrefix(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhrasePrefixQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterMatchPhrasePrefix adds a match_phrase_prefix query to filter clause
func (b *BoolBuilder) FilterMatchPhrasePrefix(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhrasePrefixQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotMatchPhrasePrefix adds a match_phrase_prefix query to must_not clause
func (b *BoolBuilder) MustNotMatchPhrasePrefix(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhrasePrefixQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- MatchPhrase Methods ---

// MustMatchPhrase adds a match_phrase query to must clause
func (b *BoolBuilder) MustMatchPhrase(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhraseQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldMatchPhrase adds a match_phrase query to should clause
func (b *BoolBuilder) ShouldMatchPhrase(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhraseQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterMatchPhrase adds a match_phrase query to filter clause
func (b *BoolBuilder) FilterMatchPhrase(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhraseQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotMatchPhrase adds a match_phrase query to must_not clause
func (b *BoolBuilder) MustNotMatchPhrase(field string, value string) *BoolBuilder {
	query, err := b.buildMatchPhraseQuery(field, value)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// --- KNN (k-Nearest Neighbor) Methods ---
// These methods are for fields mapped as "knn_vector" type (storing dense vectors for similarity search).

// MustKNN adds a basic k-NN query to must clause
// field: the knn_vector field name
// vector: the query vector (must match the field's dimension)
// k: number of nearest neighbors to return
func (b *BoolBuilder) MustKNN(field string, vector []float32, k int) *BoolBuilder {
	query := b.buildKNNQuery(field, vector, k)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldKNN adds a basic k-NN query to should clause
func (b *BoolBuilder) ShouldKNN(field string, vector []float32, k int) *BoolBuilder {
	query := b.buildKNNQuery(field, vector, k)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterKNN adds a basic k-NN query to filter clause
func (b *BoolBuilder) FilterKNN(field string, vector []float32, k int) *BoolBuilder {
	query := b.buildKNNQuery(field, vector, k)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustNotKNN adds a basic k-NN query to must_not clause
func (b *BoolBuilder) MustNotKNN(field string, vector []float32, k int) *BoolBuilder {
	query := b.buildKNNQuery(field, vector, k)
	b.query.MustNot = append(b.query.MustNot, query)
	return b
}

// MustKNNWithFilter adds a k-NN query with filter to must clause
func (b *BoolBuilder) MustKNNWithFilter(field string, vector []float32, k int, filter Query) *BoolBuilder {
	query := b.buildKNNQueryWithFilter(field, vector, k, filter)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldKNNWithFilter adds a k-NN query with filter to should clause
func (b *BoolBuilder) ShouldKNNWithFilter(field string, vector []float32, k int, filter Query) *BoolBuilder {
	query := b.buildKNNQueryWithFilter(field, vector, k, filter)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterKNNWithFilter adds a k-NN query with filter to filter clause
func (b *BoolBuilder) FilterKNNWithFilter(field string, vector []float32, k int, filter Query) *BoolBuilder {
	query := b.buildKNNQueryWithFilter(field, vector, k, filter)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustKNNWithMinScore adds a k-NN query with min score threshold to must clause
func (b *BoolBuilder) MustKNNWithMinScore(field string, vector []float32, k int, minScore float64) *BoolBuilder {
	query := b.buildKNNQueryWithMinScore(field, vector, k, minScore)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldKNNWithMinScore adds a k-NN query with min score threshold to should clause
func (b *BoolBuilder) ShouldKNNWithMinScore(field string, vector []float32, k int, minScore float64) *BoolBuilder {
	query := b.buildKNNQueryWithMinScore(field, vector, k, minScore)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterKNNWithMinScore adds a k-NN query with min score threshold to filter clause
func (b *BoolBuilder) FilterKNNWithMinScore(field string, vector []float32, k int, minScore float64) *BoolBuilder {
	query := b.buildKNNQueryWithMinScore(field, vector, k, minScore)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustKNNWithMaxDistance adds a k-NN query with max distance threshold to must clause
func (b *BoolBuilder) MustKNNWithMaxDistance(field string, vector []float32, k int, maxDistance float64) *BoolBuilder {
	query := b.buildKNNQueryWithMaxDistance(field, vector, k, maxDistance)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldKNNWithMaxDistance adds a k-NN query with max distance threshold to should clause
func (b *BoolBuilder) ShouldKNNWithMaxDistance(field string, vector []float32, k int, maxDistance float64) *BoolBuilder {
	query := b.buildKNNQueryWithMaxDistance(field, vector, k, maxDistance)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterKNNWithMaxDistance adds a k-NN query with max distance threshold to filter clause
func (b *BoolBuilder) FilterKNNWithMaxDistance(field string, vector []float32, k int, maxDistance float64) *BoolBuilder {
	query := b.buildKNNQueryWithMaxDistance(field, vector, k, maxDistance)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// MustKNNWithOptions adds a k-NN query with full options to must clause
func (b *BoolBuilder) MustKNNWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) *BoolBuilder {
	query := b.buildKNNQueryWithOptions(field, vector, k, opts)
	b.query.Must = append(b.query.Must, query)
	return b
}

// ShouldKNNWithOptions adds a k-NN query with full options to should clause
func (b *BoolBuilder) ShouldKNNWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) *BoolBuilder {
	query := b.buildKNNQueryWithOptions(field, vector, k, opts)
	b.query.Should = append(b.query.Should, query)
	return b
}

// FilterKNNWithOptions adds a k-NN query with full options to filter clause
func (b *BoolBuilder) FilterKNNWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) *BoolBuilder {
	query := b.buildKNNQueryWithOptions(field, vector, k, opts)
	b.query.Filter = append(b.query.Filter, query)
	return b
}

// --- Raw Query Methods ---

// MustQuery adds raw Query objects to must clause
func (b *BoolBuilder) MustQuery(queries ...Query) *BoolBuilder {
	b.query.Must = append(b.query.Must, queries...)
	return b
}

// ShouldQuery adds raw Query objects to should clause
func (b *BoolBuilder) ShouldQuery(queries ...Query) *BoolBuilder {
	b.query.Should = append(b.query.Should, queries...)
	return b
}

// FilterQuery adds raw Query objects to filter clause
func (b *BoolBuilder) FilterQuery(queries ...Query) *BoolBuilder {
	b.query.Filter = append(b.query.Filter, queries...)
	return b
}

// MustNotQuery adds raw Query objects to must_not clause
func (b *BoolBuilder) MustNotQuery(queries ...Query) *BoolBuilder {
	b.query.MustNot = append(b.query.MustNot, queries...)
	return b
}

// --- Configuration Methods ---

// MinimumShouldMatch sets the minimum number of should clauses that must match
func (b *BoolBuilder) MinimumShouldMatch(value interface{}) *BoolBuilder {
	b.query.MinimumShouldMatch = value
	return b
}

// --- Build Methods ---

// Build returns the constructed Query
func (b *BoolBuilder) Build() Query {
	if len(b.errors) > 0 {
		log.Printf("BoolBuilder has %d errors:", len(b.errors))
		for _, err := range b.errors {
			log.Printf("  - %v", err)
		}
	}
	return Query{Bool: &b.query}
}

// BuildWithErrors returns the Query and any errors encountered
func (b *BoolBuilder) BuildWithErrors() (Query, []error) {
	return Query{Bool: &b.query}, b.errors
}

// HasErrors returns true if errors were encountered during building
func (b *BoolBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Errors returns all errors encountered during building
func (b *BoolBuilder) Errors() []error {
	return b.errors
}

// --- Helper Methods ---

func (b *BoolBuilder) resolveField(field string, queryType QueryType) (string, error) {
	for _, indexPattern := range b.indices {
		merged, err := b.mapper.GetMergedMapping(b.ctx, indexPattern)
		if err != nil {
			log.Printf("Warning: Failed to fetch mapping for '%s': %v", indexPattern, err)
			continue
		}

		resolvedField, err := merged.ResolveFieldName(field, queryType)
		if err == nil {
			return resolvedField, nil
		}
	}
	return field, nil
}

func (b *BoolBuilder) getFieldInfo(field string) (FieldInfo, bool) {
	for _, indexPattern := range b.indices {
		merged, err := b.mapper.GetMergedMapping(b.ctx, indexPattern)
		if err != nil {
			continue
		}
		info, ok := merged.GetFieldInfo(field)
		if ok {
			return info, true
		}
	}
	return FieldInfo{}, false
}

func (b *BoolBuilder) buildTermQuery(field string, value interface{}) (Query, error) {
	resolvedField, err := b.resolveField(field, QueryTypeTerm)
	if err != nil {
		if info, ok := b.getFieldInfo(field); ok && info.Type == "text" && len(info.Fields) == 0 {
			log.Printf("Warning: Field '%s' is text type with no .keyword sub-field, converting Term to Match query", field)
			return Query{
				Match: map[string]Match{
					field: {Query: fmt.Sprint(value)},
				},
			}, nil
		}
		return Query{}, fmt.Errorf("term field '%s': %w", field, err)
	}

	return Query{
		Term: map[string]map[string]interface{}{
			resolvedField: {"value": value},
		},
	}, nil
}

func (b *BoolBuilder) buildTermsQuery(field string, values []interface{}) (Query, error) {
	resolvedField, err := b.resolveField(field, QueryTypeTerms)
	if err != nil {
		return Query{}, fmt.Errorf("terms field '%s': %w", field, err)
	}

	return Query{
		Terms: map[string][]interface{}{
			resolvedField: values,
		},
	}, nil
}

func (b *BoolBuilder) buildCIDRQuery(field string, cidr string) Query {
	// IP fields don't need .keyword resolution - use field as-is
	// OpenSearch natively supports CIDR notation in term queries on IP fields
	return Query{
		Term: map[string]map[string]interface{}{
			field: {"value": cidr},
		},
	}
}

func (b *BoolBuilder) buildCIDRsQuery(field string, cidrs []string) Query {
	// IP fields don't need .keyword resolution - use field as-is
	values := make([]interface{}, len(cidrs))
	for i, cidr := range cidrs {
		values[i] = cidr
	}
	return Query{
		Terms: map[string][]interface{}{
			field: values,
		},
	}
}

func (b *BoolBuilder) buildIPRangeContainsQuery(field string, ip string) Query {
	// For ip_range fields: find documents where the stored range contains this IP
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      ip,
				"lte":      ip,
				"relation": "contains",
			},
		},
	}
}

func (b *BoolBuilder) buildIPRangeIntersectsQuery(field string, fromIP, toIP string) Query {
	// For ip_range fields: find documents where the stored range overlaps with the query range
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "intersects",
			},
		},
	}
}

func (b *BoolBuilder) buildIPRangeWithinQuery(field string, fromIP, toIP string) Query {
	// For ip_range fields: find documents where the stored range is entirely within the query range
	return Query{
		Range: map[string]map[string]interface{}{
			field: {
				"gte":      fromIP,
				"lte":      toIP,
				"relation": "within",
			},
		},
	}
}

func (b *BoolBuilder) buildMatchQuery(field string, value string) Query {
	info, ok := b.getFieldInfo(field)
	if ok {
		if info.Type == "keyword" || (info.Type != "text" && info.AllowsTerm) {
			log.Printf("Warning: Field '%s' is %s type, converting Match to Term query", field, info.Type)
			resolvedField, _ := b.resolveField(field, QueryTypeTerm)
			return Query{
				Term: map[string]map[string]interface{}{
					resolvedField: {"value": value},
				},
			}
		}
	}

	return Query{
		Match: map[string]Match{
			field: {Query: value},
		},
	}
}

func (b *BoolBuilder) buildRangeQuery(field string, op string, value interface{}) (Query, error) {
	resolvedField, err := b.resolveField(field, QueryTypeRange)
	if err != nil {
		return Query{}, fmt.Errorf("range field '%s': %w", field, err)
	}

	return Query{
		Range: map[string]map[string]interface{}{
			resolvedField: {op: value},
		},
	}, nil
}

func (b *BoolBuilder) buildExistsQuery(field string) Query {
	return Query{
		Exists: map[string]string{"field": field},
	}
}

func (b *BoolBuilder) buildWildcardQuery(field string, pattern string) Query {
	resolvedField, _ := b.resolveField(field, QueryTypeWildcard)
	return Query{
		Wildcard: map[string]map[string]interface{}{
			resolvedField: {"value": pattern},
		},
	}
}

func (b *BoolBuilder) buildPrefixQuery(field string, prefix string) (Query, error) {
	resolvedField, err := b.resolveField(field, QueryTypePrefix)
	if err != nil {
		return Query{}, fmt.Errorf("prefix field '%s': %w", field, err)
	}

	return Query{
		Prefix: map[string]string{resolvedField: prefix},
	}, nil
}

func (b *BoolBuilder) buildFuzzyQuery(field string, value string, fuzziness ...string) (Query, error) {
	resolvedField, err := b.resolveField(field, QueryTypeFuzzy)
	if err != nil {
		// Fuzzy works like Match - needs text fields
		if info, ok := b.getFieldInfo(field); ok && info.Type == "keyword" {
			log.Printf("Warning: Field '%s' is keyword type, fuzzy queries work better on text fields", field)
		}
		return Query{}, fmt.Errorf("fuzzy field '%s': %w", field, err)
	}

	fuzzyValue := map[string]interface{}{"value": value}
	if len(fuzziness) > 0 {
		fuzzyValue["fuzziness"] = fuzziness[0]
	}

	return Query{
		Fuzzy: map[string]map[string]interface{}{
			resolvedField: fuzzyValue,
		},
	}, nil
}

func (b *BoolBuilder) buildRegexpQuery(field string, pattern string) Query {
	resolvedField, _ := b.resolveField(field, QueryTypeRegexp)
	return Query{
		Regexp: map[string]string{resolvedField: pattern},
	}
}

func (b *BoolBuilder) buildMatchPhrasePrefixQuery(field string, value string) (Query, error) {
	// match_phrase_prefix works like match - needs text fields
	info, ok := b.getFieldInfo(field)
	if ok && info.Type != "text" {
		return Query{}, fmt.Errorf("match_phrase_prefix field '%s' is %s type, requires text field", field, info.Type)
	}

	return Query{
		MatchPhrasePrefix: map[string]MatchPhrasePrefix{
			field: {Query: value},
		},
	}, nil
}

func (b *BoolBuilder) buildMatchPhraseQuery(field string, value string) (Query, error) {
	info, ok := b.getFieldInfo(field)
	if ok && info.Type != "text" {
		return Query{}, fmt.Errorf("match_phrase field '%s' is %s type, requires text field", field, info.Type)
	}

	return Query{
		MatchPhrase: map[string]MatchPhrase{
			field: {Query: value},
		},
	}, nil
}

// --- KNN Helper Methods ---

func (b *BoolBuilder) buildKNNQuery(field string, vector []float32, k int) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector: vector,
				K:      k,
			},
		},
	}
}

func (b *BoolBuilder) buildKNNQueryWithFilter(field string, vector []float32, k int, filter Query) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector: vector,
				K:      k,
				Filter: &filter,
			},
		},
	}
}

func (b *BoolBuilder) buildKNNQueryWithMinScore(field string, vector []float32, k int, minScore float64) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector:   vector,
				K:        k,
				MinScore: &minScore,
			},
		},
	}
}

func (b *BoolBuilder) buildKNNQueryWithMaxDistance(field string, vector []float32, k int, maxDistance float64) Query {
	return Query{
		KNN: map[string]*KNNQuery{
			field: {
				Vector:      vector,
				K:           k,
				MaxDistance: &maxDistance,
			},
		},
	}
}

func (b *BoolBuilder) buildKNNQueryWithOptions(field string, vector []float32, k int, opts KNNQueryOptions) Query {
	knn := &KNNQuery{
		Vector:      vector,
		K:           k,
		Filter:      opts.Filter,
		MinScore:    opts.MinScore,
		MaxDistance: opts.MaxDistance,
		Boost:       opts.Boost,
	}

	// Add method parameters if any are set
	if opts.EfSearch != nil || opts.Nprobe != nil {
		knn.MethodParameters = &KNNMethodParameters{
			EfSearch: opts.EfSearch,
			Nprobe:   opts.Nprobe,
		}
	}

	// Add rescore options if set
	if opts.Rescore != nil {
		knn.Rescore = opts.Rescore
		if opts.OversampleFactor != nil {
			knn.RescoreContext = &KNNRescoreContext{
				OversampleFactor: opts.OversampleFactor,
			}
		}
	}

	return Query{
		KNN: map[string]*KNNQuery{
			field: knn,
		},
	}
}

// --- Simple Helper Functions (no field resolution) ---

// Or creates a bool query with should clauses and minimum_should_match=1
func Or(queries ...Query) Query {
	return Query{
		Bool: &Bool{
			Should:             queries,
			MinimumShouldMatch: 1,
		},
	}
}

// And creates a bool query with must clauses
func And(queries ...Query) Query {
	return Query{
		Bool: &Bool{
			Must: queries,
		},
	}
}

// Not creates a bool query with must_not clauses
func Not(queries ...Query) Query {
	return Query{
		Bool: &Bool{
			MustNot: queries,
		},
	}
}
