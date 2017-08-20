package winrm

import (
	"fmt"
	"time"
)

// Endpoint struct holds configurations
// for the server endpoint
type Endpoint struct {
	// host name or ip address
	Host string
	// port to determine if it's http or https default
	// winrm ports (http:5985, https:5986).Versions
	// of winrm can be customized to listen on other ports
	Port int
	// set the flag true for https connections
	HTTPS bool
	// set the flag true for skipping ssl verifications
	Insecure bool
	// if set, used to verify the hostname on the returned certificate
	TLSServerName string
	// pointer pem certs, and key
	CACert []byte // cert auth to intdetify the server cert
	Key    []byte // public key for client auth connections
	Cert   []byte // cert for client auth connections
	// duration timeout for the underling tcp conn(http/https base protocol)
	// if the time exceeds the connection is cloded/timeouts
	Timeout time.Duration
}

func (ep *Endpoint) url() string {
	var scheme string
	if ep.HTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s:%d/wsman", scheme, ep.Host, ep.Port)
}

// NewEndpoint returns new pointer to struct Endpoint, with a default 60s response header timeout
func NewEndpoint(host string, port int, https bool, insecure bool, Cacert, cert, key []byte, timeout time.Duration) *Endpoint {
	endpoint := &Endpoint{
		Host:     host,
		Port:     port,
		HTTPS:    https,
		Insecure: insecure,
		CACert:   Cacert,
		Key:      key,
		Cert:     cert,
	}
	// if the timeout was set
	if timeout != 0 {
		endpoint.Timeout = timeout
	} else {
		// assign default 60sec timeout
		endpoint.Timeout = 60 * time.Second
	}

	return endpoint
}
