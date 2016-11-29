package bitbucket

import (
	"bytes"
	"net/http"
)

type BitbucketClient struct {
	Username string
	Password string
}

func (c *BitbucketClient) Get(endpoint string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.bitbucket.org/"+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)
	return client.Do(req)

}

func (c *BitbucketClient) Post(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.bitbucket.org/"+endpoint, jsonpayload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Add("content-type", "application/json")
	return client.Do(req)
}

func (c *BitbucketClient) Put(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", "https://api.bitbucket.org/"+endpoint, jsonpayload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Add("content-type", "application/json")
	return client.Do(req)
}

func (c *BitbucketClient) PutOnly(endpoint string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("PUT", "https://api.bitbucket.org/"+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return client.Do(req)
}

func (c *BitbucketClient) Delete(endpoint string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", "https://api.bitbucket.org/"+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return client.Do(req)
}
