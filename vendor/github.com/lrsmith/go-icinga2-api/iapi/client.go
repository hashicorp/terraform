// Package iapi provides a client for interacting with an Icinga2 Server
package iapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Server ... Use to be ClientConfig
type Server struct {
	Username           string
	Password           string
	BaseURL            string
	AllowUnverifiedSSL bool
	httpClient         *http.Client
}

// func New ...
func New(username, password, url string, allowUnverifiedSSL bool) (*Server, error) {
	return &Server{username, password, url, allowUnverifiedSSL, nil}, nil
}

// func Config ...
func (server *Server) Config(username, password, url string, allowUnverifiedSSL bool) (*Server, error) {

	// TODO : Add code to verify parameters
	return &Server{username, password, url, allowUnverifiedSSL, nil}, nil

}

func (server *Server) Connect() error {

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: server.AllowUnverifiedSSL,
		},
	}

	server.httpClient = &http.Client{
		Transport: t,
	}

	request, err := http.NewRequest("GET", server.BaseURL, nil)
	if err != nil {
		server.httpClient = nil
	}

	request.SetBasicAuth(server.Username, server.Password)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := server.httpClient.Do(request)
	defer response.Body.Close()

	if (err != nil) || (response.StatusCode != 200) {
		server.httpClient = nil
		fmt.Printf("Failed to connect to %s : %s\n", server.BaseURL, response.Status)
		return err
	}

	return nil

}

// NewAPIRequest ...
func (server *Server) NewAPIRequest(method, APICall string, jsonString []byte) (*APIResult, error) {

	fullURL := server.BaseURL + APICall

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: server.AllowUnverifiedSSL,
		},
	}

	server.httpClient = &http.Client{
		Transport: t,
	}

	request, requestErr := http.NewRequest(method, fullURL, bytes.NewBuffer(jsonString))
	if requestErr != nil {
		return nil, requestErr
	}

	request.SetBasicAuth(server.Username, server.Password)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	//if Debug {
	//dump, _ := httputil.DumpRequestOut(request, true)
	//fmt.Printf("HTTP Request\n%s\n", dump)
	//}

	response, doErr := server.httpClient.Do(request)
	defer response.Body.Close()

	if doErr != nil {
		return nil, doErr
	}

	var results APIResult
	if decodeErr := json.NewDecoder(response.Body).Decode(&results); decodeErr != nil {
		return nil, decodeErr
	}

	if results.Code == 0 { // results.Code has default value so set it.
		//fmt.Println("Setting Result Code")
		results.Code = response.StatusCode
	}

	if results.Status == "" { // results.Status has default value, so set it.
		//fmt.Println("Setting Result Status")
		results.Status = response.Status
	}

	//fmt.Printf("<<%v>>", results.Results)

	switch results.Code {
	case 0:
		results.ErrorString = "Did not get a response code."
	case 404:
		results.ErrorString = results.Status
	case 200:
		results.ErrorString = results.Status
	default:
		theError := strings.Replace(results.Results.([]interface{})[0].(map[string]interface{})["errors"].([]interface{})[0].(string), "\n", " ", -1)
		results.ErrorString = strings.Replace(theError, "Error: ", "", -1)

	}

	return &results, nil

}
