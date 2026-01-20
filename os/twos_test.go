package os_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/threatwinds/go-sdk/entities"
	twos "github.com/threatwinds/go-sdk/os"
)

type TestStruct struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func TestParseSource(t *testing.T) {
	source := twos.HitSource{
		"field1": "value1",
		"field2": 123,
	}

	expected := TestStruct{
		Field1: "value1",
		Field2: 123,
	}

	var result TestStruct

	err := source.ParseSource(&result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestBuildIndexPattern(t *testing.T) {
	if twos.BuildIndexPattern(twos.CommentPrefix) != "comment-*" {
		t.Error("expected comment-*")
	}

	if twos.BuildIndexPattern(twos.EntityPrefix, twos.ConsolidatedPrefix) != "entity-consolidated-*" {
		t.Error("expected entity-consolidated-*")
	}
}

func TestBuildIndex(t *testing.T) {
	date, err := time.Parse(time.RFC3339, "1993-10-21T20:54:05Z")
	if err != nil {
		t.Error(err)
	}

	gen := twos.BuildIndex(date, twos.RelationPrefix, twos.HistoryPrefix)

	if gen != "relation-history-1993-10" {
		t.Error("expected relation-history-1993-10")
	}
}

func TestConnect(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("Error connecting to OpenSearch: %v", err)
	}
}

func TestEntityWorkflow(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}
	if nodes[0] != "" {
		if strings.HasPrefix(nodes[0], "https://") {
			nodes[0] = strings.Replace(nodes[0], "https://", "http://", 1)
		} else if !strings.HasPrefix(nodes[0], "http://") {
			nodes[0] = "http://" + nodes[0]
		}
	}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("Error connecting to OpenSearch: %v", err)
		return
	}

	index := twos.BuildCurrentIndex(twos.CommentPrefix)
	hash := sha256.Sum256([]byte(index))
	entityID := hex.EncodeToString(hash[:])
	userID := uuid.MustParse("46e3c6eb-1403-4f52-94b7-9a067bb75b47")
	parentID := uuid.MustParse("3a7b0d22-9d2b-4bec-9756-ee58790e147b")
	var updatedUserID uuid.UUID

	// Clean up index to ensure fresh start
	_ = twos.DeleteIndex(context.Background(), index)

	// Step 1: Index Doc
	t.Run("IndexDoc", func(t *testing.T) {
		comment := entities.Comment{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			EntityID:  entityID,
			Comment:   "Testing",
			UserID:    userID,
			ParentID:  parentID,
			VisibleBy: []string{"public"},
		}

		err = twos.IndexDoc(context.Background(), comment, index, uuid.NewString())
		if err != nil {
			t.Error(err)
		}
		// Refresh index immediately
		err = twos.RefreshIndex(context.Background(), twos.BuildIndexPattern(twos.CommentPrefix))
		if err != nil {
			t.Error(err)
		}
	})

	// Step 2: SearchIn
	t.Run("SearchIn", func(t *testing.T) {
		req := twos.SearchRequest{
			Version:      true,
			From:         0,
			Size:         10,
			StoredFields: []string{"*"},
			Source:       &twos.Source{Excludes: []string{"visibleBy"}},
			Query: &twos.Query{
				Bool: &twos.Bool{
					Filter: []twos.Query{
						{
							Term: map[string]map[string]interface{}{
								"userID.keyword": {
									"value": userID.String(),
								},
							},
						},
					},
				},
			},
		}

		resp, err := req.SearchIn(context.Background(), []string{twos.BuildIndexPattern(twos.CommentPrefix)}, []string{"public"})
		if err != nil {
			t.Error(err)
		}

		if resp.Hits.Total.Value <= 0 {
			t.Error("entity should be found")
		}
	})

	// Step 3: Save (Update)
	t.Run("Save", func(t *testing.T) {
		req := twos.SearchRequest{
			Version:      true,
			From:         0,
			Size:         10,
			StoredFields: []string{"*"},
			// Removed Excludes: []string{"visibleBy"} to ensure visibleBy is preserved
			Query: &twos.Query{
				Bool: &twos.Bool{
					Filter: []twos.Query{
						{
							Term: map[string]map[string]interface{}{
								"userID.keyword": {
									"value": userID.String(),
								},
							},
						},
					},
				},
			},
		}

		resp, err := req.SearchIn(context.Background(), []string{twos.BuildIndexPattern(twos.CommentPrefix)}, []string{"public"})
		if err != nil {
			t.Error(err)
		}

		if resp.Hits.Total.Value <= 0 {
			t.Error("entity should be found for update")
			return
		}

		for _, hit := range resp.Hits.Hits {
			var source entities.Comment
			err = hit.Source.ParseSource(&source)
			if err != nil {
				t.Error(err)
			}

			updatedUserID = uuid.New()
			source.UserID = updatedUserID

			err = hit.Source.SetSource(source)
			if err != nil {
				t.Error(err)
			}

			err = hit.Save(context.Background())
			if err != nil {
				t.Error(err)
			}
		}
		// Refresh index immediately
		err = twos.RefreshIndex(context.Background(), twos.BuildIndexPattern(twos.CommentPrefix))
		if err != nil {
			t.Error(err)
		}
	})

	// Step 4: Delete
	t.Run("Delete", func(t *testing.T) {
		if updatedUserID == uuid.Nil {
			t.Fatal("updatedUserID was not set in Save step")
		}

		req := twos.SearchRequest{
			Version:      true,
			From:         0,
			Size:         10,
			StoredFields: []string{"*"},
			// Removed Excludes: []string{"visibleBy"}
			Query: &twos.Query{
				Bool: &twos.Bool{
					Filter: []twos.Query{
						{
							Term: map[string]map[string]interface{}{
								"userID.keyword": {
									"value": updatedUserID.String(),
								},
							},
						},
					},
				},
			},
		}

		resp, err := req.SearchIn(context.Background(), []string{twos.BuildIndexPattern(twos.CommentPrefix)}, []string{"public"})
		if err != nil {
			t.Error(err)
		}

		if resp.Hits.Total.Value <= 0 {
			t.Errorf("entity should be found for delete using new userID %s", updatedUserID)
			return
		}

		for _, hit := range resp.Hits.Hits {
			err = hit.Delete(context.Background())
			if err != nil {
				t.Error(err)
			}
		}
		// Refresh index immediately
		err = twos.RefreshIndex(context.Background(), twos.BuildIndexPattern(twos.CommentPrefix))
		if err != nil {
			t.Error(err)
		}
	})
}
