package entities

import (
	"net"
	"strings"
)

// ValidateCIDR validates if a given string is a valid CIDR notation and returns the CIDR string and its SHA3-256 hash.
func ValidateCIDR(value string) (string, string, error) {
	ip, cidr, err := net.ParseCIDR(strings.ToLower(value))
	if err != nil {
		return "", "", err
	}

	_, _, e := ValidateIP(ip.String())
	if e != nil {
		return "", "", e
	}

	cstr := cidr.String()
	return cstr, GenerateSHA3256(cstr), nil
}
