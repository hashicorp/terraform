package logentries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type LogSetClient struct {
	AccountKey string
}

type LogSetCreateRequest struct {
	Name     string
	Location string
	DistVer  string
	System   string
	DistName string
}

type LogSetCreateResponse struct {
	AgentKey string `json:"agent_key"`
	HostKey  string `json:"host_key"`
	LogSet   `json:"host"`
	ApiResponse
}

func (l *LogSetClient) Create(createRequest LogSetCreateRequest) (*LogSet, error) {
	form := url.Values{}
	form.Add("request", "register")
	form.Add("user_key", l.AccountKey)
	form.Add("name", createRequest.Name)
	form.Add("hostname", createRequest.Location)
	form.Add("distver", createRequest.DistVer)
	form.Add("system", createRequest.System)
	form.Add("distname", createRequest.DistName)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response LogSetCreateResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.Response == "ok" {
			return &response.LogSet, nil
		} else {
			return nil, fmt.Errorf("failed to create log %s: %s", createRequest.Name, response.ResponseReason)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve log %s: %s", createRequest.Name, string(body))
}

type LogSetReadRequest struct {
	Key string
}

type LogSetReadResponse struct {
	LogSet LogSet
	ApiResponse
}

func (l *LogSetClient) Read(readRequest LogSetReadRequest) (*LogSet, error) {
	userClient := NewUserClient(l.AccountKey)
	response, err := userClient.Read(UserReadRequest{})
	if err != nil {
		return nil, err
	}

	for _, logSet := range response.LogSets {
		if logSet.Key == readRequest.Key {
			return &logSet, nil
		}
	}

	return nil, fmt.Errorf("No such log set with key %s", readRequest.Key)
}

type LogSetUpdateRequest struct {
	Key      string
	Name     string
	Location string
}

type LogSetUpdateResponse struct {
	AgentKey string `json:"agent_key"`
	Key      string `json:"host_key"`
	LogSet   `json:"host"`
	ApiResponse
}

func (l *LogSetClient) Update(updateRequest LogSetUpdateRequest) (*LogSet, error) {
	form := url.Values{}
	form.Add("request", "set_host")
	form.Add("user_key", l.AccountKey)
	form.Add("host_key", updateRequest.Key)
	form.Add("name", updateRequest.Name)
	form.Add("hostname", string(updateRequest.Location))
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response LogSetUpdateResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.Response == "ok" {
			return &response.LogSet, nil
		} else {
			return nil, fmt.Errorf("failed to update log set %s: %s", updateRequest.Name, response.ResponseReason)
		}
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve log set %s: %s", updateRequest.Name, string(body))
}

type LogSetDeleteRequest struct {
	Key string
}

type LogSetDeleteResponse struct {
	UserKey string `json:"user_key"`
	Key     string `json:"host_key"`
	Worker  string `json:"worker"`
	ApiResponse
}

func (l *LogSetClient) Delete(deleteRequest LogSetDeleteRequest) error {
	form := url.Values{}
	form.Add("request", "rm_host")
	form.Add("user_key", l.AccountKey)
	form.Add("host_key", deleteRequest.Key)
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return err
	}

	if resp.StatusCode == 200 {
		var deleteResponse LogSetDeleteResponse
		json.NewDecoder(resp.Body).Decode(&deleteResponse)
		if deleteResponse.Response == "ok" {
			return nil
		} else {
			return fmt.Errorf("failed to delete log set %s: %s", deleteResponse.Key, deleteResponse.ResponseReason)
		}
	}

	return fmt.Errorf("failed to delete log set %s: %s", deleteRequest.Key, resp.Body)
}

func NewLogSetClient(account_key string) *LogSetClient {
	logset := &LogSetClient{AccountKey: account_key}
	return logset
}
