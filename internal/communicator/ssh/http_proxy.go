package ssh

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// Dialer implements for SSH over HTTP Proxy.
type Dialer struct {
	proxy proxyInfo
	// forwarding Dialer
	forward proxy.Dialer
}

type proxyInfo struct {
	// HTTP Proxy host or host:port
	host string
	// HTTP Proxy scheme
	scheme string
	// User name if http proxy needs authentication
	username string
	// User password if http proxy needs authentication
	password string
	// Whether the HTTP Proxy requires authentication
	auth bool
}

func newProxyInfo(host, scheme, username, password string) *proxyInfo {
	p := &proxyInfo{
		host:   host,
		scheme: scheme,
	}

	if username != "" && password != "" {
		p.auth = true
		p.username = username
		p.password = password
	}

	if p.scheme == "" {
		p.scheme = "http"
	}

	return p
}

func (p *proxyInfo) url() (*url.URL, error) {
	base := p.scheme + "://"

	if p.auth {
		base = base + p.username + ":" + p.password + "@"
	}

	return url.Parse(base + p.host)
}

func (p *Dialer) Dial(network, addr string) (net.Conn, error) {
	// Dial the proxy host
	c, err := p.forward.Dial(network, p.proxy.host)

	if err != nil {
		return nil, err
	}

	err = c.SetDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		return nil, err
	}

	// Generate request URL to host accessed through the proxy
	reqUrl, err := url.Parse("http://" + addr)
	if err != nil {
		c.Close()
		return nil, err
	}
	reqUrl.Scheme = ""

	// Create a request object using the CONNECT method to instruct the proxy server to tunnel a protocol other than HTTP.
	req, err := http.NewRequest("CONNECT", reqUrl.String(), nil)
	if err != nil {
		c.Close()
		return nil, err
	}

	// If http proxy requires authentication, configure settings for basic authentication.
	if p.proxy.auth {
		req.SetBasicAuth(p.proxy.username, p.proxy.password)
		req.Header.Add("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(p.proxy.username+":"+p.proxy.password)))
	}

	// Do not close the connection after sending this request and reading its response.
	req.Close = false

	// Writes the request in the form expected by an HTTP proxy.
	err = req.Write(c)
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

// NewHttpProxyDialer generate Http Proxy Dialer
func NewHttpProxyDialer(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	var proxyUserName, proxyPassword string
	if u.User != nil {
		proxyUserName = u.User.Username()
		proxyPassword, _ = u.User.Password()
	}

	pd := &Dialer{
		proxy:   *newProxyInfo(u.Host, u.Scheme, proxyUserName, proxyPassword),
		forward: forward,
	}

	return pd, nil
}

// RegisterDialerType register schemes used by `proxy.FromURL`
func RegisterDialerType() {
	proxy.RegisterDialerType("http", NewHttpProxyDialer)
	proxy.RegisterDialerType("https", NewHttpProxyDialer)
}

// NewHttpProxyConn create a connection to connect through the proxy server.
func NewHttpProxyConn(p *proxyInfo, targetAddr string) (net.Conn, error) {
	proxyURL, err := p.url()
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
