package opensearch

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateNd(t *testing.T) {
	t.Run("Generate Index Bulk", func(t *testing.T) {
		nd, err := generateNd([]BulkItem{
			{
				Index: "event-2021-01-01",
				Id:    uuid.NewString(),
				Body: []byte(`{"field": "value1"}`),
				Action: "index",
			},
			{
				Index: "event-2021-01-01",
				Id:    uuid.NewString(),
				Body: []byte(`{"field": "value2"}`),
				Action: "index",
			},
			{
				Index: "event-2021-01-01",
				Id:    uuid.NewString(),
				Body: []byte(`{"field": "value3"}`),
				Action: "index",
			},
			{
				Index: "event-2021-01-01",
				Id:    uuid.NewString(),
				Body: []byte(`{"field": "value4"}`),
				Action: "index",
			},
		})
		if err != nil {
			t.Errorf("Expected nil, got %s", err)
		}

		t.Log(nd)
	})

}
