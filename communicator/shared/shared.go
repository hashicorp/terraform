package shared

import (
	"fmt"
	"net"
)

// IpFormat formats the IP correctly, so we don't provide IPv6 address in an IPv4 format during node communication. We return the ip parameter as is if it's an IPv4 address or a hostname.
func IpFormat(ip string) string {
	ipObj := net.ParseIP(ip)
	// Return the ip/host as is if it's either a hostname or an IPv4 address.
	if ipObj == nil || ipObj.To4() != nil {
		return ip
	}

	return fmt.Sprintf("[%s]", ip)
}
