package entities

import (
	"fmt"
	"net"
	"strings"
)

// ValidateIP validates if the given IP address is valid and not private, multicast, loopback, or unspecified.
// It returns the validated IP address and its SHA3-256 hash.
func ValidateIP(value interface{}) (string, string, error) {
	v, ok := value.(string)
	if !ok {
		return "", "", fmt.Errorf("value not string: %v", value)
	}

	addr := net.ParseIP(strings.ToLower(v))
	if addr == nil {
		return "", "", fmt.Errorf("invalid IP: %s", v)
	}
	if addr.IsPrivate() {
		return "", "", fmt.Errorf("cannot accept private IP: %s", v)
	}
	if addr.IsInterfaceLocalMulticast() {
		return "", "", fmt.Errorf("cannot accept interface local multicast IP: %s", v)
	}
	if addr.IsLinkLocalMulticast() {
		return "", "", fmt.Errorf("cannot accept link local multicast IP: %s", v)
	}
	if addr.IsLinkLocalUnicast() {
		return "", "", fmt.Errorf("cannot accept link local unicast IP: %s", v)
	}
	if addr.IsLoopback() {
		return "", "", fmt.Errorf("cannot accept loopback IP: %s", v)
	}
	if addr.IsMulticast() {
		return "", "", fmt.Errorf("cannot accept multicast IP: %s", v)
	}
	if addr.IsUnspecified() {
		return "", "", fmt.Errorf("cannot accept unspecified IP: %s", v)
	}

	a := addr.String()

	return a, GenerateSHA3256(a), nil
}
