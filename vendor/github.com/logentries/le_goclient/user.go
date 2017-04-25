package logentries

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type UserClient struct {
	UserKey string
}

type UserReadRequest struct {
}

type UserReadResponse struct {
	User
	ApiResponse
}

func (u *UserClient) Read(readRequest UserReadRequest) (*UserReadResponse, error) {
	form := url.Values{}
	form.Add("request", "get_user")
	form.Add("load_hosts", "1")
	form.Add("load_logs", "1")
	form.Add("load_alerts", "0")
	form.Add("user_key", u.UserKey)
	form.Add("id", "terraform")
	resp, err := http.PostForm("https://api.logentries.com/", form)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		var response UserReadResponse
		json.NewDecoder(resp.Body).Decode(&response)
		return &response, nil
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("Could not retrieve account info: %s", string(body))
}

func NewUserClient(account_key string) *UserClient {
	account := UserClient{UserKey: account_key}
	return &account
}
