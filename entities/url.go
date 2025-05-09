package entities

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateURL validates a given URL string and returns the URL in lowercase and its SHA3-256 hash.
// If the value is not a string, it returns an error.
func ValidateURL(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value is not string: %v", value)
	}

	tmp, err := url.ParseRequestURI(v)
	if err != nil {
		return "", "", err
	}
	tmp.Host = strings.ToLower(tmp.Host)
	tmp.Scheme = strings.ToLower(tmp.Scheme)

	surl := tmp.String()
	return surl, GenerateSHA3256(surl), nil
}
