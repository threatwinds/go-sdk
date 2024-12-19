package go_sdk

import (
	"context"
	"net"
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
		return "", Error("failed to create Dial context", err, nil)
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", Error("failed to get local address", nil, nil)
	}

	if localAddr.IP == nil {
		return "", Error("failed to get local IP address", nil, nil)
	}

	return localAddr.IP.String(), nil
}
