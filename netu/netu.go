package netu

import (
	"net"
	"strings"
)

func GetIPv4Str() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127_0_0_0"
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		// Skip loopback and non-IPv4 addresses
		if ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		// Replace dots with underscores
		return strings.ReplaceAll(ipnet.IP.String(), ".", "_")
	}

	return "127_0_0_0"
}
