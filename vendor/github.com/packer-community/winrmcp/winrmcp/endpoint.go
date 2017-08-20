package winrmcp

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/masterzen/winrm"
)

func parseEndpoint(addr string, https bool, insecure bool, tlsServerName string, caCert []byte, timeout time.Duration) (*winrm.Endpoint, error) {
	var host string
	var port int

	if addr == "" {
		return nil, errors.New("Couldn't convert \"\" to an address.")
	}
	if !strings.Contains(addr, ":") || (strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]")) {
		host = addr
		port = 5985
	} else {
		shost, sport, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("Couldn't convert \"%s\" to an address.", addr)
		}
		// Check for IPv6 addresses and reformat appropriately
		host = IpFormat(shost)
		port, err = strconv.Atoi(sport)
		if err != nil {
			return nil, errors.New("Couldn't convert \"%s\" to a port number.")
		}
	}

	return &winrm.Endpoint{
		Host:          host,
		Port:          port,
		HTTPS:         https,
		Insecure:      insecure,
		TLSServerName: tlsServerName,
		CACert:        caCert,
		Timeout:       timeout,
	}, nil
}

// IpFormat formats the IP correctly, so we don't provide IPv6 address in an IPv4 format during node communication.
// We return the ip parameter as is if it's an IPv4 address or a hostname.
func IpFormat(ip string) string {
	ipObj := net.ParseIP(ip)
	// Return the ip/host as is if it's either a hostname or an IPv4 address.
	if ipObj == nil || ipObj.To4() != nil {
		return ip
	}

	return fmt.Sprintf("[%s]", ip)
}
