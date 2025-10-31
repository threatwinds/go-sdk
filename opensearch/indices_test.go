package opensearch

import (
	"fmt"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestBuildTenantIndexPattern(t *testing.T) {
	t.Run("Build Tenant Index Pattern", func(t *testing.T) {
		id := uuid.New()
		expected := fmt.Sprintf("%s-event-*", id.String())
		generated := BuildTenantIndexPattern(id, "event")
		if expected != generated {
			t.Errorf("Expected %s, got %s", expected, generated)
		}
	})

	t.Run("Build Tenant Index", func(t *testing.T) {
		id := uuid.New()
		now := time.Now().UTC()
		expected := fmt.Sprintf("%s-event-%s", id.String(), now.Format("2006-01-02"))
		generated := BuildTenantIndex(id, now, "event")
		if expected != generated {
			t.Errorf("Expected %s, got %s", expected, generated)
		}
	})
}
