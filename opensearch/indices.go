package opensearch

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// BuildCurrentTenantIndex returns a string representing the current index based on the given elements,
// including tenant ID.
func BuildCurrentTenantIndex(tenant uuid.UUID, elements ...string) string {
	return BuildTenantIndex(tenant, time.Now().UTC(), elements...)
}

// BuildCurrentIndex returns a string representing the current index based on the given elements.
func BuildCurrentIndex(elements ...string) string {
	return BuildIndex(time.Now().UTC(), elements...)
}

// BuildTenantIndex returns a string representing an index based on the given date and elements, including tenant ID.
func BuildTenantIndex(tenant uuid.UUID, date time.Time, elements ...string) string {
	var index = make([]string, 0, 3)
	index = append(index, tenant.String())
	index = append(index, elements...)

	return BuildIndex(date, index...)
}

func BuildIndex(date time.Time, elements ...string) string {
	elements = append(elements, date.Format("2006-01-02"))
	return strings.Join(elements, "-")
}

// BuildTenantIndexPattern returns a string representing an index pattern based on the given elements.
func BuildTenantIndexPattern(tenant uuid.UUID, elements ...string) string {
	var pattern = make([]string, 0, 3)
	pattern = append(pattern, tenant.String())
	pattern = append(pattern, elements...)

	return BuildIndexPattern(pattern...)
}

// BuildIndexPattern returns a string representing an index pattern based on the given elements.
func BuildIndexPattern(elements ...string) string {
	elements = append(elements, "*")
	return strings.Join(elements, "-")
}
