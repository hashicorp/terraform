package winrmcp

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/masterzen/winrm/winrm"
)

func parseEndpoint(addr string, https bool, insecure bool, caCert []byte) (*winrm.Endpoint, error) {
	var host string
	var port int

	if addr == "" {
		return nil, errors.New("Couldn't convert \"\" to an address.")
	}
	if !strings.Contains(addr, ":") {
		host = addr
		port = 5985
	} else {
		shost, sport, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Couldn't convert \"%s\" to an address.", addr))
		}
		host = shost
		port, err = strconv.Atoi(sport)
		if err != nil {
			return nil, errors.New("Couldn't convert \"%s\" to a port number.")
		}
	}

	return &winrm.Endpoint{
		Host:     host,
		Port:     port,
		HTTPS:    https,
		Insecure: insecure,
		CACert:   &caCert,
	}, nil
}
