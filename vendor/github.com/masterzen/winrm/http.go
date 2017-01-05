package winrm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/masterzen/winrm/soap"
)

var soapXML = "application/soap+xml"

// HttpPost type func for handling http requests
type HttpPost func(*Client, *soap.SoapMessage) (string, error)

// body func reads the response body and return it as a string
func body(response *http.Response) (string, error) {

	// if we recived the content we expected
	if strings.Contains(response.Header.Get("Content-Type"), "application/soap+xml") {
		body, err := ioutil.ReadAll(response.Body)
		defer response.Body.Close()
		if err != nil {
			return "", fmt.Errorf("error while reading request body %s", err)
		}

		return string(body), nil
	}

	return "", fmt.Errorf("invalid content type")
}

// PostRequest make post to the winrm soap service
func PostRequest(client *Client, request *soap.SoapMessage) (string, error) {
	httpClient := &http.Client{Transport: client.transport}

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
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http error: %d - %s", resp.StatusCode, body)
	}

	return body, err
}
