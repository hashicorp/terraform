package shield

import (
	"bytes"
	"crypto/tls"
	"net/http"
)

type ShieldClient struct {
	ServerUrl string
	Username  string
	Password  string
	Insecure  bool
}

func (c *ShieldClient) Get(endpoint string) (*http.Response, error) {

	client := &http.Client{}
	if c.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}

	req, err := http.NewRequest("GET", "https://"+c.ServerUrl+"/"+endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)

	return client.Do(req)

}

func (c *ShieldClient) Post(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	client := &http.Client{}
	if c.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}

	req, err := http.NewRequest("POST", "https://"+c.ServerUrl+"/"+endpoint, jsonpayload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Add("content-type", "application/json")
	return client.Do(req)
}

func (c *ShieldClient) Put(endpoint string, jsonpayload *bytes.Buffer) (*http.Response, error) {
	client := &http.Client{}
	if c.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}

	req, err := http.NewRequest("PUT", "https://"+c.ServerUrl+"/"+endpoint, jsonpayload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Add("content-type", "application/json")
	return client.Do(req)
}

func (c *ShieldClient) PutOnly(endpoint string) (*http.Response, error) {
	client := &http.Client{}
	if c.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	req, err := http.NewRequest("PUT", "https://"+c.ServerUrl+"/"+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return client.Do(req)
}

func (c *ShieldClient) Delete(endpoint string) (*http.Response, error) {
	client := &http.Client{}
	if c.Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}

	req, err := http.NewRequest("DELETE", "https://"+c.ServerUrl+"/"+endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	return client.Do(req)
}
