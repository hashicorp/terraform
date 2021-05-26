package ssh

import (
	"bufio"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
)

// httpProxyDialer implements for SSH over HTTP Proxy.
type httpProxyDialer struct {
	// HTTP Proxy hostname and address
	host string
	// forwarding Dialer
	forward proxy.Dialer
	// Whether the HTTP Proxy requires authentication
	auth bool
	// User name if http proxy needs authentication
	username string
	// User password if http proxy needs authentication
	password string
}

// Dial directory invokes net.Dial with the supplied parameters
func (p *httpProxyDialer) Dial(network, addr string) (net.Conn, error) {
	// Dial the proxy host
	c, err := p.forward.Dial(network, p.host)

	if err != nil {
		return nil, err
	}

	// Generate request URL to host accessed through the proxy
	reqUrl, err := url.Parse("http://" + addr)
	if err != nil {
		c.Close()
		return nil, err
	}

	// Create a request object using the CONNECT method to instruct the proxy server to tunnel a protocol other than HTTP.
	req, err := http.NewRequest("CONNECT", reqUrl.String(), nil)
	if err != nil {
		c.Close()
		return nil, err
	}

	// If http proxy requires authentication, configure settings for basic authentication.
	if p.auth {
		req.SetBasicAuth(p.username, p.password)
	}

	// Do not close the connection after sending this request and reading its response.
	req.Close = false

	// Writes the request in the form expected by an HTTP proxy.
	err = req.WriteProxy(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	res, err := http.ReadResponse(bufio.NewReader(c), req)

	if err != nil {
		res.Body.Close()
		c.Close()
		return nil, err
	}

	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		c.Close()
		return nil, fmt.Errorf("Connection Error: StatusCode: %d", res.StatusCode)
	}

	return c, nil
}

// Generate Http Proxy Dialer
func NewHttpProxyDialer(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	host := u.Host
	p := &httpProxyDialer{
		host:    host,
		forward: forward,
	}

	if u.User != nil {
		p.auth = true
		p.username = u.User.Username()
		p.password, _ = u.User.Password()
	}

	return p, nil
}

// Registered schemes are used by `proxy.FromURL`
func RegisterDialerType() {
	proxy.RegisterDialerType("http", NewHttpProxyDialer)
	proxy.RegisterDialerType("https", NewHttpProxyDialer)
}

// Create a connection to connect through the proxy server.
func NewHttpProxyConn(proxyAddr, targetAddr string) (net.Conn, error) {
	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		return nil, err
	}

	proxyDialer, err := proxy.FromURL(proxyURL, proxy.Direct)

	if err != nil {
		return nil, err
	}

	proxyConn, err := proxyDialer.Dial("tcp", targetAddr)

	if err != nil {
		return nil, err
	}

	return proxyConn, err
}
