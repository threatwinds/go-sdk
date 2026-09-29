package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// GetMainIP retrieves the main IP address of the local machine by establishing
// a UDP connection to a remote server (Google's public DNS server in this case).
// It returns the IP address as a string and an error if any error occurs
// during the process.
//
// Returns:
//   - string: The main IP address of the local machine.
//   - error: An error object if there is an issue obtaining the IP address.
func GetMainIP() (string, error) {
	// Add context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to create Dial context: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("failed to get local address: invalid type")
	}

	if localAddr.IP == nil {
		return "", fmt.Errorf("failed to get local IP address: IP is nil")
	}

	return localAddr.IP.String(), nil
}

// CheckConnectivity checks if a given URL is reachable within the specified timeout.
// It sends a HEAD request to minimize bandwidth usage.
//
// The URL is whatever the caller wants reached — a ThreatWinds service, a
// vendor's endpoint, a captive-portal probe — so no error header is read: this
// function has no way to know it is talking to a peer that runs
// catcher.GinError, and adopting an x-error-id on the strength of the header
// name alone would let any endpoint name one of this org's occurrences. Both
// failures below therefore carry a locally generated id instead, which is
// honest about where the id came from and still gives an operator one value to
// grep by. A HEAD response has no body to read a richer detail from in any case.
func CheckConnectivity(url string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Head(url)
	if err != nil {
		return withGeneratedErrorID(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return withGeneratedErrorID(fmt.Errorf("server returned status: %s", resp.Status))
	}

	return nil
}
