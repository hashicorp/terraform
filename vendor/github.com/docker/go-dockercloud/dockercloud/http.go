package dockercloud

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
)

var customUserAgent = "go-dockercloud/" + version
var jar http.CookieJar

func SetUserAgent(name string) string {
	customUserAgent = ""
	customUserAgent = name + " go-dockercloud/" + version
	return customUserAgent
}

func SetBaseUrl() string {
	if os.Getenv("DOCKERCLOUD_REST_HOST") != "" {
		BaseUrl = os.Getenv("DOCKERCLOUD_REST_HOST")
		BaseUrl = BaseUrl + "/api/"
	}
	return BaseUrl
}

func init() {
	BaseUrl = SetBaseUrl()
	jar, _ = cookiejar.New(nil)
}

func DockerCloudCall(url string, requestType string, requestBody []byte) ([]byte, error) {
	if !IsAuthenticated() {
		err := LoadAuth()
		if err != nil {
			return nil, err
		}
	}
	client := &http.Client{Jar: jar}

	req, err := http.NewRequest(requestType, BaseUrl+url, bytes.NewBuffer(requestBody))

	req.Header.Add("Authorization", AuthHeader)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", customUserAgent)

	if Debug == true {
		log.Printf("req: %s", string(requestBody))
	}

	response, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if response.StatusCode > 300 {
		return nil, fmt.Errorf("Failed API call: %s", response.Status)
	}

	jar.SetCookies(req.URL, response.Cookies())

	data, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	if Debug == true {
		log.Printf("res: %s", string(data))
	}
	return data, nil
}
