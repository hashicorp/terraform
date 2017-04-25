package logentries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type LogClient struct {
	AccountKey string
}

type LogCreateRequest struct {
	LogSetKey string
	Name      string
	Retention string
	Source    string
	Type      string
	Filename  string
}

type LogCreateResponse struct {
	Key    string `json:"log_key"`
	Worker string `json:"worker"`
	Log    Log    `json:"log"`
	ApiResponse
}

func (l *LogClient) Create(createRequest LogCreateRequest) (*Log, error) {
	form := url.Values{}
	form.Add("request", "new_log")
	form.Add("user_key", l.AccountKey)
	form.Add("host_key", createRequest.LogSetKey)
	form.Add("name", createRequest.Name)
	form.Add("type", createRequest.Type)
	form.Add("filename", createRequest.Filename)
	form.Add("retention", createRequest.Retention)
	form.Add("source", createRequest.Source)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response LogCreateResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.Response == "ok" {
			return &response.Log, nil
		} else {
			return nil, fmt.Errorf("failed to create log %s: %s", createRequest.Name, response.ResponseReason)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve log %s: %s", createRequest.Name, string(body))
}

type LogReadRequest struct {
	LogSetKey string
	Key       string
}

type LogReadResponse struct {
	Log Log `json:"log"`
	ApiResponse
}

func (l *LogClient) Read(readRequest LogReadRequest) (*Log, error) {
	form := url.Values{}
	form.Add("request", "get_log")
	form.Add("log_key", readRequest.Key)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response LogReadResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.Response == "ok" {
			return &response.Log, nil
		} else {
			return nil, fmt.Errorf("failed to get log %s: %s", readRequest.Key, response.ResponseReason)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve log %s: %s", readRequest.Key, string(body))
}

type LogUpdateRequest struct {
	Key       string
	Name      string
	Type      string
	Source    string
	Retention string
	Filename  string
}

type LogUpdateResponse struct {
	Key    string `json:"log_key"`
	Worker string `json:"worker"`
	Log    Log    `json:"log"`
	ApiResponse
}

func (l *LogClient) Update(updateRequest LogUpdateRequest) (*Log, error) {
	form := url.Values{}
	form.Add("request", "set_log")
	form.Add("user_key", l.AccountKey)
	form.Add("log_key", updateRequest.Key)
	form.Add("name", updateRequest.Name)
	form.Add("type", updateRequest.Type)
	form.Add("source", updateRequest.Source)
	form.Add("filename", updateRequest.Filename)
	form.Add("retention", updateRequest.Retention)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response LogUpdateResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.Response == "ok" {
			return &response.Log, nil
		} else {
			return nil, fmt.Errorf("failed to update log %s: %s", updateRequest.Name, response.ResponseReason)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve log %s: %s", updateRequest.Name, string(body))
}

type LogDeleteRequest struct {
	LogSetKey string
	Key       string
}

type LogDeleteResponse struct {
	LogSetKey string `json:"host_key"`
	UserKey   string `json:"user_key"`
	Key       string `json:"log_key"`
	Worker    string `json:"worker"`
	ApiResponse
}

func (l *LogClient) Delete(deleteRequest LogDeleteRequest) error {
	form := url.Values{}
	form.Add("request", "rm_log")
	form.Add("user_key", l.AccountKey)
	form.Add("host_key", deleteRequest.LogSetKey)
	form.Add("log_key", deleteRequest.Key)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		var deleteResponse LogDeleteResponse
		json.NewDecoder(resp.Body).Decode(&deleteResponse)
		if deleteResponse.Response == "ok" {
			return nil
		} else {
			return fmt.Errorf("failed to delete log %s: %s", deleteResponse.Key, deleteResponse.ResponseReason)
		}
	}

	return fmt.Errorf("failed to delete log %s: %s", deleteRequest.Key, resp.Body)
}

func NewLogClient(account_key string) *LogClient {
	log := &LogClient{AccountKey: account_key}
	return log
}
