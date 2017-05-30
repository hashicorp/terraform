package gonsx

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"github.com/sky-uk/gonsx/api"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// NewNSXClient  Creates a new nsxclient object.
func NewNSXClient(url string, user string, password string, ignoreSSL bool, debug bool) *NSXClient {
	nsxClient := new(NSXClient)
	nsxClient.URL = url
	nsxClient.User = user
	nsxClient.Password = password
	nsxClient.IgnoreSSL = ignoreSSL
	nsxClient.debug = debug
	return nsxClient
}

// NSXClient struct.
type NSXClient struct {
	URL       string
	User      string
	Password  string
	IgnoreSSL bool
	debug     bool
}

// Do - makes the API call.
func (nsxClient *NSXClient) Do(api api.NSXApi) error {
	requestURL := fmt.Sprintf("%s%s", nsxClient.URL, api.Endpoint())

	var requestPayload io.Reader
	if api.RequestObject() != nil {
		requestXMLBytes, marshallingErr := xml.Marshal(api.RequestObject())
		if marshallingErr != nil {
			log.Fatal(marshallingErr)
		}
		if nsxClient.debug {
			log.Println(string(requestXMLBytes))
		}
		requestPayload = bytes.NewReader(requestXMLBytes)
	}
	if nsxClient.debug {
		log.Println("requestURL:", requestURL)
	}
	req, err := http.NewRequest(api.Method(), requestURL, requestPayload)
	if err != nil {
		log.Println("ERROR building the request: ", err)
		return err
	}

	req.SetBasicAuth(nsxClient.User, nsxClient.Password)
	// TODO: remove this hardcoded value!
	req.Header.Set("Content-Type", "application/xml")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: nsxClient.IgnoreSSL},
	}
	httpClient := &http.Client{Transport: tr}
	res, err := httpClient.Do(req)
	if err != nil {
		log.Println("ERROR executing request: ", err)
		return err
	}
	defer res.Body.Close()
	return nsxClient.handleResponse(api, res)
}

func (nsxClient *NSXClient) handleResponse(api api.NSXApi, res *http.Response) error {
	api.SetStatusCode(res.StatusCode)
	bodyText, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("ERROR reading response: ", err)
		return err
	}

	api.SetRawResponse(bodyText)

	if nsxClient.debug {
		log.Println(string(bodyText))
	}

	if isXML(res.Header.Get("Content-Type")) && api.StatusCode() == 200 {
		xmlerr := xml.Unmarshal(bodyText, api.ResponseObject())
		if xmlerr != nil {
			log.Println("ERROR unmarshalling response: ", err)
			return err
		}
	} else {
		api.SetResponseObject(string(bodyText))
	}
	return nil
}

func isXML(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "/xml")
}
