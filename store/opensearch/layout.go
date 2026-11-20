package opensearch

import (
	"fmt"
	"strings"
	"time"

	"github.com/threatwinds/go-sdk/store"
)

// Layout maps datasets to physical indices. The application supplies one.
type Layout interface {
	ReadIndices(s store.Scope) ([]string, error)
	WriteIndex(s store.Scope, now time.Time) (string, error)
	Datasets() []store.Dataset
	DataTypeOf(ds store.Dataset, index string) string
}

// DatasetSpec describes one dataset as "<prefix>[-<dataType>][-<date>]".
type DatasetSpec struct {
	Prefix          string
	IncludeDataType bool
	// DateFormat is a Go time layout; empty means the index does not roll.
	DateFormat string
}

// SpecLayout covers any prefix-and-date naming scheme without writing code.
type SpecLayout map[store.Dataset]DatasetSpec

var _ Layout = SpecLayout{}

func (l SpecLayout) spec(ds store.Dataset) (DatasetSpec, error) {
	s, ok := l[ds]
	if !ok {
		return DatasetSpec{}, fmt.Errorf("%w: dataset %q is not in the layout", store.ErrUnsupported, ds)
	}
	if s.Prefix == "" {
		return DatasetSpec{}, fmt.Errorf("opensearch: dataset %q has no prefix", ds)
	}
	return s, nil
}

func (l SpecLayout) ReadIndices(s store.Scope) ([]string, error) {
	spec, err := l.spec(s.Dataset)
	if err != nil {
		return nil, err
	}
	name := spec.Prefix
	if spec.IncludeDataType && s.DataType != "" {
		name += "-" + s.DataType
	}
	if spec.DateFormat != "" || (spec.IncludeDataType && s.DataType == "") {
		name += "-*"
	}
	return []string{name}, nil
}

func (l SpecLayout) WriteIndex(s store.Scope, now time.Time) (string, error) {
	spec, err := l.spec(s.Dataset)
	if err != nil {
		return "", err
	}
	name := spec.Prefix
	if spec.IncludeDataType {
		if s.DataType == "" {
			return "", fmt.Errorf("opensearch: writing to %s requires a DataType", s.Dataset)
		}
		name += "-" + s.DataType
	}
	if spec.DateFormat != "" {
		name += "-" + now.UTC().Format(spec.DateFormat)
	}
	return name, nil
}

func (l SpecLayout) Datasets() []store.Dataset {
	out := make([]store.Dataset, 0, len(l))
	for ds := range l {
		out = append(out, ds)
	}
	return out
}

// DataTypeOf strips the prefix and the segments the date format contributes.
func (l SpecLayout) DataTypeOf(ds store.Dataset, index string) string {
	spec, err := l.spec(ds)
	if err != nil || !spec.IncludeDataType {
		return ""
	}
	rest, ok := strings.CutPrefix(index, spec.Prefix+"-")
	if !ok {
		return ""
	}
	dateSegments := 0
	if spec.DateFormat != "" {
		dateSegments = strings.Count(spec.DateFormat, "-") + 1
	}
	parts := strings.Split(rest, "-")
	if len(parts) <= dateSegments {
		return rest
	}
	return strings.Join(parts[:len(parts)-dateSegments], "-")
}

// DataTypeOfPattern is the reverse of ReadIndices: given an index pattern as a
// v11 rule writes it ("v11-log-deceptive-bytes-*"), it names the dataType that
// pattern reads. A pattern covering every dataType ("v11-log-*") reports an
// empty one. ok is false for a pattern that is not of this dataset's shape — a
// different prefix, or a wildcard anywhere but the end — so that an unknown
// pattern is refused rather than read as something it is not.
func (l SpecLayout) DataTypeOfPattern(ds store.Dataset, pattern string) (dataType string, ok bool) {
	spec, err := l.spec(ds)
	if err != nil {
		return "", false
	}

	rest, found := strings.CutPrefix(pattern, spec.Prefix)
	if !found {
		return "", false
	}
	// The prefix has to end where a name segment ends, or "v11-logger-*" would
	// read as the dataType "ger" of "v11-log".
	if rest != "" && rest[0] != '-' && rest[0] != '*' {
		return "", false
	}
	// A wildcard glued to the prefix ("v11-log*") reads every dataType too.
	rest = strings.TrimSuffix(strings.TrimPrefix(rest, "-"), "*")
	rest = strings.TrimSuffix(rest, "-")

	if strings.ContainsAny(rest, "*?") {
		return "", false
	}
	if rest == "" {
		return "", true
	}
	if !spec.IncludeDataType {
		return "", false
	}
	return rest, true
}
