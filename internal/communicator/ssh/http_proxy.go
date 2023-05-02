// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ssh

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// Dialer implements for SSH over HTTP Proxy.
type proxyDialer struct {
	proxy proxyInfo
	// forwarding Dialer
	forward proxy.Dialer
}

type proxyInfo struct {
	// HTTP Proxy host or host:port
	host string
	// HTTP Proxy scheme
	scheme string
	// An immutable encapsulation of username and password details for a URL
	userInfo *url.Userinfo
}

func newProxyInfo(host, scheme, username, password string) *proxyInfo {
	p := &proxyInfo{
		host:   host,
		scheme: scheme,
	}

	p.userInfo = url.UserPassword(username, password)

	if p.scheme == "" {
		p.scheme = "http"
	}

	return p
}

func (p *proxyInfo) url() *url.URL {
	return &url.URL{
		Scheme: p.scheme,
		User:   p.userInfo,
		Host:   p.host,
	}
}

func (p *proxyDialer) Dial(network, addr string) (net.Conn, error) {
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
	reqUrl := &url.URL{
		Scheme: "",
		Host:   addr,
	}

	// Create a request object using the CONNECT method to instruct the proxy server to tunnel a protocol other than HTTP.
	req, err := http.NewRequest("CONNECT", reqUrl.String(), nil)
	if err != nil {
		c.Close()
		return nil, err
	}

	// If http proxy requires authentication, configure settings for basic authentication.
	if p.proxy.userInfo.String() != "" {
		username := p.proxy.userInfo.Username()
		password, _ := p.proxy.userInfo.Password()
		req.SetBasicAuth(username, password)
		req.Header.Add("Proxy-Authorization", req.Header.Get("Authorization"))
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
func newHttpProxyDialer(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	var proxyUserName, proxyPassword string
	if u.User != nil {
		proxyUserName = u.User.Username()
		proxyPassword, _ = u.User.Password()
	}

	pd := &proxyDialer{
		proxy:   *newProxyInfo(u.Host, u.Scheme, proxyUserName, proxyPassword),
		forward: forward,
	}

	return pd, nil
}

// RegisterDialerType register schemes used by `proxy.FromURL`
func RegisterDialerType() {
	proxy.RegisterDialerType("http", newHttpProxyDialer)
	proxy.RegisterDialerType("https", newHttpProxyDialer)
}

// NewHttpProxyConn create a connection to connect through the proxy server.
func newHttpProxyConn(p *proxyInfo, targetAddr string) (net.Conn, error) {
	pd, err := proxy.FromURL(p.url(), proxy.Direct)

	if err != nil {
		return nil, err
	}

	proxyConn, err := pd.Dial("tcp", targetAddr)

	if err != nil {
		return nil, err
	}

	return proxyConn, err
}
