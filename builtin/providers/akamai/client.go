package akamai

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
)

const apiPath = "papi/v0"

type Client struct {
    Config *edgegrid.Config
}

func (c *Client) Get(endpoint string, responseStruct interface{}) error {
    client := &http.Client{}

    req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/%s/%s", c.Config.Host, apiPath, endpoint), nil)
    if err != nil {
        return err
    }

    req = edgegrid.AddRequestHeader(*c.Config, req)
    resp, err := client.Do(req)
    if err != nil {
        return err
    }

    bodyContents, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusOK {
        akamaiError, err := NewAkamaiError(bodyContents)
        if err != nil {
            return err
        }
        return akamaiError
    }
    if err := json.Unmarshal(bodyContents, responseStruct); err != nil {
        return err
    }
    return nil
}

/*
func (c *Client) Post(endpoint string, payload *bytes.Buffer) (*http.Response, error) {
    client := &http.Client{}

    req, err := http.NewRequest("POST", fmt.Sprintf("https://%s/%s/%s", c.Config.Host, apiPath, endpoint), payload)
    if err != nil {
        return nil, err
    }

    req = edgegrid.AddRequestHeader(*c.Config, req)
    resp, err := client.Do(req)
    if err != nil {
        return err
    }

    if resp.StatusCode != http.StatusCreated {
        akamaiError := &AkamaiError{}
        if err := json.Unmarshal(bodyContents, &akamaiError); err != nil {
            return err
        }
        akamaiError.RequestBody = string(body)
        akamaiError.ResponseBody = string(bodyContents)
        return akamaiError
    }
    if err := json.Unmarshal(bodyContents, responseStruct); err != nil {
        return err
    }
    return nil
}
*/
