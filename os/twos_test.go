package os_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
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

func TestIndexDoc(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("Error connecting to OpenSearch: %v", err)
	}

	index := twos.BuildCurrentIndex(twos.CommentPrefix)
	hash := sha256.Sum256([]byte(index))
	entityID := hex.EncodeToString(hash[:])

	comment := entities.Comment{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		EntityID:  entityID,
		Comment:   "Testing",
		UserID:    uuid.MustParse("46e3c6eb-1403-4f52-94b7-9a067bb75b47"),
		ParentID:  uuid.MustParse("3a7b0d22-9d2b-4bec-9756-ee58790e147b"),
		VisibleBy: []string{"public"},
	}

	err = twos.IndexDoc(context.Background(), comment, index, uuid.NewString())
	if err != nil {
		t.Error(err)
	}
}

func TestSearchIn(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("error connecting to OpenSearch: %v", err)
	}

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
								"value": "46e3c6eb-1403-4f52-94b7-9a067bb75b47",
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
}

func TestSave(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("Error connecting to OpenSearch: %v", err)
	}

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
								"value": "46e3c6eb-1403-4f52-94b7-9a067bb75b47",
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

	for _, hit := range resp.Hits.Hits {
		var source entities.Comment
		err = hit.Source.ParseSource(&source)
		if err != nil {
			t.Error(err)
		}

		source.UserID = uuid.New()

		err = hit.Source.SetSource(source)
		if err != nil {
			t.Error(err)
		}

		err = hit.Save(context.Background())
		if err != nil {
			t.Error(err)
		}
	}
}

func TestDelete(t *testing.T) {
	nodes := []string{os.Getenv("NODES")}

	err := twos.Connect(nodes, os.Getenv("USER"), os.Getenv("PASSWORD"))
	if err != nil {
		t.Errorf("Error connecting to OpenSearch: %v", err)
	}

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
							"parentID.keyword": {
								"value": "3a7b0d22-9d2b-4bec-9756-ee58790e147b",
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

	for _, hit := range resp.Hits.Hits {
		var source entities.Comment
		err = hit.Source.ParseSource(&source)
		if err != nil {
			t.Error(err)
		}

		source.UserID = uuid.New()

		err = hit.Source.SetSource(source)
		if err != nil {
			t.Error(err)
		}

		err = hit.Delete(context.Background())
		if err != nil {
			t.Error(err)
		}
	}
}
