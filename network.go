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
		return "", Error(Trace(), map[string]interface{}{
			"cause": err.Error(),
			"error": "error: failed to create Dial context",
		})
	}
	defer func() {
		_ = conn.Close()
	}()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", Error(Trace(), map[string]interface{}{
			"error": "error: failed to cast LocalAddr to UDPAddr",
		})
	}

	if localAddr.IP == nil {
		return "", Error(Trace(), map[string]interface{}{
			"error": "failed to get IP address",
		})
	}

	return localAddr.IP.String(), nil
}
