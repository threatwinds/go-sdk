package entities

import (
	"fmt"
	"net"
	"strings"
)

// ValidateIP validates if the given IP address is valid and not private, multicast, loopback, or unspecified.
// It returns the validated IP address and its SHA3-256 hash.
func ValidateIP(value string) (string, string, error) {
	addr := net.ParseIP(strings.ToLower(value))
	if addr == nil {
		return "", "", fmt.Errorf("invalid IP: %s", value)
	}
	if addr.IsPrivate() {
		return "", "", fmt.Errorf("cannot accept private IP: %s", value)
	}
	if addr.IsInterfaceLocalMulticast() {
		return "", "", fmt.Errorf("cannot accept interface local multicast IP: %s", value)
	}
	if addr.IsLinkLocalMulticast() {
		return "", "", fmt.Errorf("cannot accept link local multicast IP: %s", value)
	}
	if addr.IsLinkLocalUnicast() {
		return "", "", fmt.Errorf("cannot accept link local unicast IP: %s", value)
	}
	if addr.IsLoopback() {
		return "", "", fmt.Errorf("cannot accept loopback IP: %s", value)
	}
	if addr.IsMulticast() {
		return "", "", fmt.Errorf("cannot accept multicast IP: %s", value)
	}
	if addr.IsUnspecified() {
		return "", "", fmt.Errorf("cannot accept unspecified IP: %s", value)
	}

	a := addr.String()

	return a, GenerateSHA3256(a), nil
}
