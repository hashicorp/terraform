package icinga2

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Config struct to store Icinga2 Provider connection information.
type Config struct {
	APIURL      string
	APIUser     string
	APIPassword string
}

func (c *Config) loadAndValidate() error {

	if c.APIURL == "" {
		return fmt.Errorf("Invalid endpoint type provided : %s", c.APIURL)
	}

	return nil

}

// Client blah blah
func (c *Config) Client(httpMethod string, endPoint string, jsonStr []byte) (int, interface{}, error) {

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	url := fmt.Sprintf("%s/%s", c.APIURL, endPoint)
	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return 500, nil, err
	}

	req.SetBasicAuth(c.APIUser, c.APIPassword)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: transport}
	response, err := client.Do(req)
	defer response.Body.Close()

	if err == nil {

		respBody, _ := ioutil.ReadAll(response.Body)
		var rsp interface{}
		if unmarshalErr := json.Unmarshal(respBody, &rsp); unmarshalErr != nil {
			log.Fatal(unmarshalErr)
		}
		return response.StatusCode, rsp, err
	}

	return 500, nil, err
}
