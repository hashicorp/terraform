package edgegrid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func authenticate(c Client, req *http.Request) *http.Request {
	auth := Auth(NewAuthParams(req, c.GetCredentials().AccessToken, c.GetCredentials().ClientToken, c.GetCredentials().ClientSecret))

	req.Header.Add("Authorization", auth)

	return req
}

func concat(arr []string) string {
	var buff bytes.Buffer

	for _, elem := range arr {
		buff.WriteString(elem)
	}

	return buff.String()
}

func urlPathWithQuery(req *http.Request) string {
	var query string

	if req.URL.RawQuery != "" {
		query = concat([]string{
			"?",
			req.URL.RawQuery,
		})
	} else {
		query = ""
	}

	return concat([]string{
		req.URL.Path,
		query,
	})
}

func resourceRequest(c Client, method string, url string, body []byte, responseStruct interface{}) error {
	if LogRequests() {
		fmt.Printf("Request url: \n\t%s\nrequest body: \n\t%s \n\n", url, string(body))
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(body))

	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	authReq := authenticate(c, req)
	resp, err := c.GetHTTPClient().Do(authReq)

	if err != nil {
		return err
	}

	bodyContents, err := ioutil.ReadAll(resp.Body)
	if LogRequests() {
		fmt.Printf("Response status: \n\t%d\nresponse body: \n\t%s \n\n", resp.StatusCode, bodyContents)
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		akError := &AkamaiError{}
		if err := json.Unmarshal(bodyContents, &akError); err != nil {
			return err
		}
		akError.RequestBody = string(body)
		akError.ResponseBody = string(bodyContents)
		return akError
	}
	if err := json.Unmarshal(bodyContents, responseStruct); err != nil {
		return err
	}
	return nil
}

func doClientReq(c Client, method string, url string, body []byte) (*http.Response, error) {
	if LogRequests() {
		fmt.Printf("Request url: \n\t%s\nrequest body: \n\t%s \n\n", url, string(body))
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(body))

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	authReq := authenticate(c, req)
	resp, err := c.GetHTTPClient().Do(authReq)

	return resp, err
}

func getXML(c Client, url string) (*http.Response, error) {
	if LogRequests() {
		fmt.Printf("Request url: \n\t%s\n", url)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "text/xml")
	authReq := authenticate(c, req)
	resp, err := c.GetHTTPClient().Do(authReq)

	return resp, err
}
