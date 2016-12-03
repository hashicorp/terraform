package net

import (
	_ "crypto/sha512" // #82254112: http://bridge.grumpy-troll.org/2014/05/golang-tls-comodo/
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"golang.org/x/net/websocket"
)

//go:generate counterfeiter . HTTPClientInterface

type HTTPClientInterface interface {
	RequestDumperInterface

	Do(*http.Request) (*http.Response, error)
	ExecuteCheckRedirect(req *http.Request, via []*http.Request) error
}

type client struct {
	*http.Client
	dumper RequestDumper
}

var NewHTTPClient = func(tr *http.Transport, dumper RequestDumper) HTTPClientInterface {
	c := client{
		&http.Client{
			Transport: tr,
		},
		dumper,
	}
	c.CheckRedirect = c.checkRedirect

	return &c
}

func (cl *client) ExecuteCheckRedirect(req *http.Request, via []*http.Request) error {
	return cl.CheckRedirect(req, via)
}

func (cl *client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > 1 {
		return errors.New(T("stopped after 1 redirect"))
	}

	prevReq := via[len(via)-1]
	cl.copyHeaders(prevReq, req, getBaseDomain(req.URL.String()) == getBaseDomain(via[0].URL.String()))
	cl.dumper.DumpRequest(req)

	return nil
}

func (cl *client) copyHeaders(from *http.Request, to *http.Request, sameDomain bool) {
	for key, values := range from.Header {
		// do not copy POST-specific headers
		if key != "Content-Type" && key != "Content-Length" && !(!sameDomain && key == "Authorization") {
			to.Header.Set(key, strings.Join(values, ","))
		}
	}
}

func (cl *client) DumpRequest(req *http.Request) {
	cl.dumper.DumpRequest(req)
}

func (cl *client) DumpResponse(res *http.Response) {
	cl.dumper.DumpResponse(res)
}

func WrapNetworkErrors(host string, err error) error {
	var innerErr error
	switch typedErr := err.(type) {
	case *url.Error:
		innerErr = typedErr.Err
	case *websocket.DialError:
		innerErr = typedErr.Err
	}

	if innerErr != nil {
		switch typedInnerErr := innerErr.(type) {
		case x509.UnknownAuthorityError:
			return errors.NewInvalidSSLCert(host, T("unknown authority"))
		case x509.HostnameError:
			return errors.NewInvalidSSLCert(host, T("not valid for the requested host"))
		case x509.CertificateInvalidError:
			return errors.NewInvalidSSLCert(host, "")
		case *net.OpError:
			if typedInnerErr.Op == "dial" {
				return fmt.Errorf("%s: %s\n%s", T("Error performing request"), err.Error(), T("TIP: If you are behind a firewall and require an HTTP proxy, verify the https_proxy environment variable is correctly set. Else, check your network connection."))
			}
		}
	}

	return fmt.Errorf("%s: %s", T("Error performing request"), err.Error())
}

func getBaseDomain(host string) string {
	hostURL, _ := url.Parse(host)
	hostStrs := strings.Split(hostURL.Host, ".")
	return hostStrs[len(hostStrs)-2] + "." + hostStrs[len(hostStrs)-1]
}
