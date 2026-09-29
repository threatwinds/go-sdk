package entities

import (
	"github.com/google/uuid"
)

type EntityConsolidated struct {
	ID              *string    `json:"id,omitempty" example:"ip-ad0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	Timestamp       string     `json:"@timestamp" example:"2021-09-29T15:59:59.000Z"`
	LastSeen        string     `json:"lastSeen" example:"2021-09-29T15:59:59.000Z"`
	Type            string     `json:"type" example:"ip"`
	Reputation      int        `json:"reputation" example:"-3"`
	BestReputation  int        `json:"bestReputation" example:"-1"`
	WorstReputation int        `json:"worstReputation" example:"-3"`
	Accuracy        int        `json:"accuracy" example:"3"`
	Attributes      Attributes `json:"attributes"`
	Tags            []string   `json:"tags" example:"[\"web-server\",\"mail-server\"]"`
	VisibleBy       []string   `json:"visibleBy" example:"[\"public\",\"quantfall\"]"`
	WellKnown       bool       `json:"wellKnown" example:"false"`
}

type EntityHistory struct {
	ID         *uuid.UUID `json:"id,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Timestamp  string     `json:"@timestamp" example:"2021-09-29T15:59:59.000Z"`
	EntityID   string     `json:"entityID" example:"ip-ad0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	Type       string     `json:"type" example:"ip"`
	UserID     uuid.UUID  `json:"userID" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Reputation int        `json:"reputation" example:"-3"`
	Attributes Attributes `json:"attributes"`
	Tags       []string   `json:"tags" example:"[\"web-server\",\"mail-server\"]"`
	VisibleBy  []string   `json:"visibleBy" example:"[\"public\",\"quantfall\"]"`
	WellKnown  bool       `json:"wellKnown" example:"false"`
}

type RelationConsolidated struct {
	ID              *string  `json:"id,omitempty" example:"ad0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	Timestamp       string   `json:"@timestamp" example:"2021-09-29T15:59:59.000Z"`
	LastSeen        string   `json:"lastSeen" example:"2021-09-29T15:59:59.000Z"`
	EntityID        string   `json:"entityID" example:"ip-fe0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	RelatedEntityID string   `json:"relatedEntityID" example:"domain-da0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	Mode            string   `json:"mode" example:"aggregation"`
	VisibleBy       []string `json:"visibleBy" example:"[\"public\",\"quantfall\"]"`
}

type RelationHistory struct {
	ID              *uuid.UUID `json:"id,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Timestamp       string     `json:"@timestamp" example:"2021-09-29T15:59:59.000Z"`
	RelationID      string     `json:"relationID" example:"ad0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	EntityID        string     `json:"entityID" example:"ip-fe0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	RelatedEntityID string     `json:"relatedEntityID" example:"domain-da0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	UserID          uuid.UUID  `json:"userID" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Mode            string     `json:"mode" example:"association"`
	VisibleBy       []string   `json:"visibleBy" example:"[\"public\",\"quantfall\"]"`
}

type Comment struct {
	ID        *string   `json:"id,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	Timestamp string    `json:"@timestamp" example:"2021-09-29T15:59:59.000Z"`
	EntityID  string    `json:"entityID" example:"ip-fe0c2ed9a0a9b23822e5907b0d009bcaf8f969db793cd1d94c40e17e0287c04b"`
	Comment   string    `json:"comment" example:"This is a comment"`
	UserID    uuid.UUID `json:"userID" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	ParentID  uuid.UUID `json:"parentID,omitempty" example:"f47ac10b-58cc-4372-a567-0e02b2c3d479"`
	VisibleBy []string  `json:"visibleBy" example:"[\"public\",\"quantfall\"]"`
}

type Entity struct {
	Type         string              `json:"type"  example:"object"`
	Attributes   Attributes          `json:"attributes"`
	Associations []EntityAssociation `json:"associations"`
	Reputation   int                 `json:"reputation" example:"-1"`
	Correlate    []string            `json:"correlate" example:"[\"md5\", \"sha1\", \"sha256\", \"sha3-256\"]"`
	Tags         []string            `json:"tags" example:"[\"malware\", \"common-file\"]"`
	VisibleBy    []string            `json:"visibleBy" example:"[\"public\"]"`
}

type EntityAssociation struct {
	Mode string `json:"mode" example:"aggregation"`
	Entity
}
