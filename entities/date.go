package entities

import (
	"time"
)

// ValidateDate validates a date string in the format "2006-01-02" and returns the formatted date string and its SHA3-256 hash.
func ValidateDate(value string) (string, string, error) {
	tmp, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", "", err
	}

	ftime := tmp.Format("2006-01-02")
	return ftime, GenerateSHA3256(ftime), nil
}

// ValidateDatetime validates a datetime string in the RFC3339Nano format and returns the formatted datetime string and its SHA3-256 hash.
func ValidateDatetime(value string) (string, string, error) {
	tmp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", "", err
	}

	ftime := tmp.Format(time.RFC3339Nano)
	return ftime, GenerateSHA3256(ftime), nil
}
