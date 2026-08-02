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
