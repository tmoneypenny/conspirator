package encoding

import (
	"net"
)

func removePortFromClientIP(host string) string {
	ip, _, err := net.SplitHostPort(host)
	if err != nil || ip == "" {
		return host
	}

	return ip
}
