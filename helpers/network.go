package helpers

import (
	"net"

	"github.com/threatwinds/logger"
)

func GetMainIP() (string, *logger.Error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", Logger().ErrorF("error getting main IP: %s", err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
