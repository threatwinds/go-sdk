package go_sdk

import (
	"net"

	"github.com/threatwinds/logger"
)

// GetMainIP retrieves the main IP address of the local machine by establishing
// a UDP connection to a remote server (Google's public DNS server in this case).
// It returns the IP address as a string and a logger.Error if any error occurs
// during the process.
//
// Returns:
//   - string: The main IP address of the local machine.
//   - *logger.Error: An error object if there is an issue obtaining the IP address.
func GetMainIP() (string, *logger.Error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", Logger().ErrorF("error getting main IP: %s", err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
