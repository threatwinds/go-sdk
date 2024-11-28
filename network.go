package go_sdk

import (
	"fmt"
	"net"
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
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("error getting main IP: %s", err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
