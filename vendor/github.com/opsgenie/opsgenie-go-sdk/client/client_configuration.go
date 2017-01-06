package client

import (
	"fmt"
	"time"
)

// ProxyConfiguration is the type that contains the proxy configurations of the OpsGenieClient.
type ProxyConfiguration struct {
	Host     string
	Port     int
	Username string
	Password string
	ProxyURI string
	Protocol string
}

// HTTPTransportSettings is the type that contains the HTTP transport layer configurations of the OpsGenieClient.
type HTTPTransportSettings struct {
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
	MaxRetryAttempts  int
}

// toString is an internal method that formats and returns proxy configurations of the OpsGenieClient.
func (proxy *ProxyConfiguration) toString() string {
	if proxy.ProxyURI != "" {
		return proxy.ProxyURI
	}
	if proxy.Protocol == "" {
		proxy.Protocol = "http"
	}
	if proxy.Username != "" && proxy.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", proxy.Protocol, proxy.Username, proxy.Password, proxy.Host, proxy.Port)
	}
	return fmt.Sprintf("%s://%s:%d", proxy.Protocol, proxy.Host, proxy.Port)
}
