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
func CheckConnectivity(url string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	return nil
}
