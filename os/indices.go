// Package twos provides functionality for building index patterns and names.
package os

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// EntityPrefix is the prefix for entity indices.
	EntityPrefix string = "entity"
	// RelationPrefix is the prefix for relationship indices.
	RelationPrefix string = "relation"
	// CommentPrefix is the prefix for commentary indices.
	CommentPrefix string = "comment"
	// ConsolidatedPrefix is the prefix for consolidated indices.
	ConsolidatedPrefix string = "consolidated"
	// HistoryPrefix is the prefix for historic indices.
	HistoryPrefix string = "history"
)

// BuildIndexPattern returns a string representing an index pattern based on the given elements.
func BuildIndexPattern(elements ...string) string {
	elements = append(elements, "*")
	return strings.Join(elements, "-")
}

// BuildCurrentIndex returns a string representing the current index based on the given elements.
func BuildCurrentIndex(elements ...string) string {
	return BuildIndex(time.Now().UTC(), elements...)
}

// BuildIndex returns a string representing an index based on the given date and elements.
func BuildIndex(date time.Time, elements ...string) string {
	elements = append(elements, date.Format("2006-01"))
	return strings.Join(elements, "-")
}

// rolloverRegex matches rollover index names like "prefix-000001"
var rolloverRegex = regexp.MustCompile(`^(.+)-(\d{6})$`)

// BuildRolloverIndex returns a rollover index name with the given sequence number.
// Example: BuildRolloverIndex("messages", 1) returns "messages-000001"
func BuildRolloverIndex(prefix string, seq int) string {
	return fmt.Sprintf("%s-%06d", prefix, seq)
}

// BuildInitialRolloverIndex returns the first rollover index name for a prefix.
// Example: BuildInitialRolloverIndex("messages") returns "messages-000001"
func BuildInitialRolloverIndex(prefix string) string {
	return BuildRolloverIndex(prefix, 1)
}

// ParseRolloverIndex parses a rollover index name and returns its components.
// Example: ParseRolloverIndex("messages-000001") returns ("messages", 1, nil)
func ParseRolloverIndex(indexName string) (prefix string, seq int, err error) {
	matches := rolloverRegex.FindStringSubmatch(indexName)
	if matches == nil {
		return "", 0, fmt.Errorf("invalid rollover index name format: %s", indexName)
	}

	seq, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid sequence number in index name: %s", indexName)
	}

	return matches[1], seq, nil
}

// NextRolloverIndex returns the next rollover index name in sequence.
// Example: NextRolloverIndex("messages-000001") returns ("messages-000002", nil)
func NextRolloverIndex(currentIndex string) (string, error) {
	prefix, seq, err := ParseRolloverIndex(currentIndex)
	if err != nil {
		return "", err
	}

	return BuildRolloverIndex(prefix, seq+1), nil
}

// IsRolloverIndex checks if an index name follows the rollover naming convention.
func IsRolloverIndex(indexName string) bool {
	return rolloverRegex.MatchString(indexName)
}

// GetRolloverSequence returns the sequence number from a rollover index name.
// Returns 0 if the index name doesn't follow the rollover convention.
func GetRolloverSequence(indexName string) int {
	_, seq, err := ParseRolloverIndex(indexName)
	if err != nil {
		return 0
	}
	return seq
}
