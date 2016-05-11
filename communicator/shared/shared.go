package shared

import (
	"fmt"
	"net"
)

// HostFormatter helps out with host formatting.
// For instance, the host parameter differs when connecting to an IPv4 address vs. an IPv6 address.
type HostFormatter interface {
	Format(string) string
}

// HostFormatterImpl implements the HostFormatter interface.
type HostFormatterImpl struct{}

// Format formats the host/IP correctly, so we don't provide IPv6 address in an IPv4 format during node communication.
func (h *HostFormatterImpl) Format(host string) string {
	ip := net.ParseIP(host)
	// Return the host as is if it's either a hostname or an IPv4 address.
	if ip == nil || ip.To4() != nil {
		return host
	}

	return fmt.Sprintf("[%s]", host)
}
