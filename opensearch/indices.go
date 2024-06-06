package opensearch

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	EventSubstr    string = "events"
	AlertSubstr    string = "alerts"
	RelationSubstr string = "relations"
)

// BuildIndexPattern returns a string representing an index pattern based on the given elements.
func BuildIndexPattern(tenant uuid.UUID, elements ...string) string {
	elements = append(elements, "*")

	var pattern = make([]string, 0, 3)
	pattern = append(pattern, tenant.String())
	pattern = append(pattern, elements...)

	return strings.Join(pattern, "-")
}

// BuildCurrentIndex returns a string representing the current index based on the given elements.
func BuildCurrentIndex(tenant uuid.UUID, elements ...string) string {
	return BuildIndex(tenant, time.Now().UTC(), elements...)
}

// BuildIndex returns a string representing an index based on the given date and elements.
func BuildIndex(tenant uuid.UUID, date time.Time, elements ...string) string {
	elements = append(elements, date.Format("2006-01-02"))

	var index = make([]string, 0, 3)
	index = append(index, tenant.String())
	index = append(index, elements...)

	return strings.Join(index, "-")
}
