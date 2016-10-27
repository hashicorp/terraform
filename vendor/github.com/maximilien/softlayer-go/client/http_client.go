package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/maximilien/softlayer-go/common"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const NON_VERBOSE = "NON_VERBOSE"

type HttpClient struct {
	HTTPClient *http.Client

	username string
	password string

	useHttps bool

	apiUrl string

	nonVerbose bool

	templatePath string
}

func NewHttpsClient(username, password, apiUrl, templatePath string) *HttpClient {
	return NewHttpClient(username, password, apiUrl, templatePath, true)
}

func NewHttpClient(username, password, apiUrl, templatePath string, useHttps bool) *HttpClient {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	hClient := &HttpClient{
		username: username,
		password: password,

		useHttps: useHttps,

		apiUrl: apiUrl,

		templatePath: filepath.Join(pwd, templatePath),

		HTTPClient: http.DefaultClient,

		nonVerbose: checkNonVerbose(),
	}

	return hClient
}

// Public methods

func (slc *HttpClient) DoRawHttpRequestWithObjectMask(path string, masks []string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	url := fmt.Sprintf("%s://%s:%s@%s/%s", slc.scheme(), slc.username, slc.password, slc.apiUrl, path)

	url += "?objectMask="
	for i := 0; i < len(masks); i++ {
		url += masks[i]
		if i != len(masks)-1 {
			url += ";"
		}
	}

	return slc.makeHttpRequest(url, requestType, requestBody)
}

func (slc *HttpClient) DoRawHttpRequestWithObjectFilter(path string, filters string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	url := fmt.Sprintf("%s://%s:%s@%s/%s", slc.scheme(), slc.username, slc.password, slc.apiUrl, path)
	url += "?objectFilter=" + filters

	return slc.makeHttpRequest(url, requestType, requestBody)
}

func (slc *HttpClient) DoRawHttpRequestWithObjectFilterAndObjectMask(path string, masks []string, filters string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	url := fmt.Sprintf("%s://%s:%s@%s/%s", slc.scheme(), slc.username, slc.password, slc.apiUrl, path)

	url += "?objectFilter=" + filters

	url += "&objectMask=filteredMask["
	for i := 0; i < len(masks); i++ {
		url += masks[i]
		if i != len(masks)-1 {
			url += ";"
		}
	}
	url += "]"

	return slc.makeHttpRequest(url, requestType, requestBody)
}

func (slc *HttpClient) DoRawHttpRequest(path string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	url := fmt.Sprintf("%s://%s:%s@%s/%s", slc.scheme(), slc.username, slc.password, slc.apiUrl, path)
	return slc.makeHttpRequest(url, requestType, requestBody)
}

func (slc *HttpClient) GenerateRequestBody(templateData interface{}) (*bytes.Buffer, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	bodyTemplate := template.Must(template.ParseFiles(filepath.Join(cwd, slc.templatePath)))
	body := new(bytes.Buffer)
	bodyTemplate.Execute(body, templateData)

	return body, nil
}

func (slc *HttpClient) HasErrors(body map[string]interface{}) error {
	if errString, ok := body["error"]; !ok {
		return nil
	} else {
		return errors.New(errString.(string))
	}
}

func (slc *HttpClient) CheckForHttpResponseErrors(data []byte) error {
	var decodedResponse map[string]interface{}
	err := json.Unmarshal(data, &decodedResponse)
	if err != nil {
		return err
	}

	if err := slc.HasErrors(decodedResponse); err != nil {
		return err
	}

	return nil
}

// Private methods

func (slc *HttpClient) scheme() string {
	if !slc.useHttps {
		return "http"
	}

	return "https"
}

func (slc *HttpClient) makeHttpRequest(url string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	req, err := http.NewRequest(requestType, url, requestBody)
	if err != nil {
		return nil, 0, err
	}

	bs, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, 0, err
	}

	if !slc.nonVerbose {
		fmt.Fprintf(os.Stderr, "\n---\n[softlayer-go] Request:\n%s\n", hideCredentials(string(bs)))
	}

	var resp *http.Response
	SL_API_WAIT_TIME, err := strconv.Atoi(os.Getenv("SL_API_WAIT_TIME"))
	if err != nil || SL_API_WAIT_TIME == 0 {
		SL_API_WAIT_TIME = 1
	}
	SL_API_RETRY_COUNT, err := strconv.Atoi(os.Getenv("SL_API_RETRY_COUNT"))
	if err != nil || SL_API_RETRY_COUNT == 0 {
		SL_API_RETRY_COUNT = 3
	}

	for i := 1; i <= SL_API_RETRY_COUNT; i++ {
		resp, err = slc.HTTPClient.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[softlayer-go] Error: %s, retrying %d time(s)\n", err.Error(), i)
			if !strings.Contains(err.Error(), "i/o timeout") && !strings.Contains(err.Error(), "connection refused") || i >= SL_API_RETRY_COUNT {
				return nil, 520, err
			}
		} else {
			break
		}

		time.Sleep(time.Duration(SL_API_WAIT_TIME) * time.Second)
	}
	defer resp.Body.Close()

	bs, err = httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if !slc.nonVerbose {
		fmt.Fprintf(os.Stderr, "[softlayer-go] Response:\n%s\n", hideCredentials(string(bs)))
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}

	if common.IsHttpErrorCode(resp.StatusCode) {
		//Try to parse response body since SoftLayer could return meaningful error message
		err = slc.CheckForHttpResponseErrorsSilently(responseBody)
		if err != nil {
			return nil, resp.StatusCode, err
		}
	}

	return responseBody, resp.StatusCode, nil
}

// Private functions

func (slc *HttpClient) CheckForHttpResponseErrorsSilently(data []byte) error {
	var decodedResponse map[string]interface{}
	parseErr := json.Unmarshal(data, &decodedResponse)
	if parseErr == nil {
		return slc.HasErrors(decodedResponse)
	}

	return nil
}

func hideCredentials(s string) string {
	hiddenStr := "\"password\":\"******\""
	r := regexp.MustCompile(`"password":"[^"]*"`)

	return r.ReplaceAllString(s, hiddenStr)
}

func checkNonVerbose() bool {
	slGoNonVerbose := os.Getenv(NON_VERBOSE)
	switch slGoNonVerbose {
	case "yes":
		return true
	case "YES":
		return true
	case "true":
		return true
	case "TRUE":
		return true
	}

	return false
}
