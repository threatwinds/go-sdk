package entities

import (
	"net/url"
	"strings"
)

// ValidateURL validates a given URL string and returns the URL in lowercase and its SHA3-256 hash.
// If the value is not a string, it returns an error.
func ValidateURL(value string) (string, string, error) {
	tmp, err := url.ParseRequestURI(value)
	if err != nil {
		return "", "", err
	}
	tmp.Host = strings.ToLower(tmp.Host)
	tmp.Scheme = strings.ToLower(tmp.Scheme)

	surl := tmp.String()
	return surl, GenerateSHA3256(surl), nil
}
