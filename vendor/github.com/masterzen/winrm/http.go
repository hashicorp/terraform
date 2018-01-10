package winrm

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/masterzen/winrm/soap"
)

var soapXML = "application/soap+xml"

// body func reads the response body and return it as a string
func body(response *http.Response) (string, error) {

	// if we recived the content we expected
	if strings.Contains(response.Header.Get("Content-Type"), "application/soap+xml") {
		body, err := ioutil.ReadAll(response.Body)
		defer func() {
			// defer can modify the returned value before
			// it is actually passed to the calling statement
			if errClose := response.Body.Close(); errClose != nil && err == nil {
				err = errClose
			}
		}()
		if err != nil {
			return "", fmt.Errorf("error while reading request body %s", err)
		}

		return string(body), nil
	}

	return "", fmt.Errorf("invalid content type")
}

type clientRequest struct {
	transport http.RoundTripper
}

func (c *clientRequest) Transport(endpoint *Endpoint) error {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: endpoint.Insecure,
			ServerName:         endpoint.TLSServerName,
		},
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		ResponseHeaderTimeout: endpoint.Timeout,
	}

	if endpoint.CACert != nil && len(endpoint.CACert) > 0 {
		certPool, err := readCACerts(endpoint.CACert)
		if err != nil {
			return err
		}

		transport.TLSClientConfig.RootCAs = certPool
	}

	c.transport = transport

	return nil
}

// Post make post to the winrm soap service
func (c clientRequest) Post(client *Client, request *soap.SoapMessage) (string, error) {
	httpClient := &http.Client{Transport: c.transport}

	req, err := http.NewRequest("POST", client.url, strings.NewReader(request.String()))
	if err != nil {
		return "", fmt.Errorf("impossible to create http request %s", err)
	}
	req.Header.Set("Content-Type", soapXML+";charset=UTF-8")
	req.SetBasicAuth(client.username, client.password)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("unknown error %s", err)
	}

	body, err := body(resp)
	if err != nil {
		return "", fmt.Errorf("http response error: %d - %s", resp.StatusCode, err.Error())
	}

	// if we have different 200 http status code
	// we must replace the error
	defer func() {
		if resp.StatusCode != 200 {
			body, err = "", fmt.Errorf("http error %d: %s", resp.StatusCode, body)
		}
	}()

	return body, err
}
