/*
Package runscope implements a client library for the runscope api (https://www.runscope.com/docs/api)

 */
package runscope

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
)

// Bucket resources are a simple way to organize your requests and tests. See https://www.runscope.com/docs/api/buckets and https://www.runscope.com/docs/buckets
type Bucket struct {
	Name           string  `json:"name,omitempty"`
	Key            string  `json:"key,omitempty"`
	Default        bool    `json:"default,omitempty"`
	AuthToken      string  `json:"auth_token,omitempty"`
	TestsURL       string  `json:"tests_url,omitempty" mapstructure:"tests_url"`
	CollectionsURL string  `json:"collections_url,omitempty"`
	MessagesURL    string  `json:"messages_url,omitempty"`
	TriggerURL     string  `json:"trigger_url,omitempty"`
	VerifySsl      bool    `json:"verify_ssl,omitempty"`
	Team           *Team   `json:"team,omitempty"`
}

// CreateBucket creates a new bucket resource. See https://www.runscope.com/docs/api/buckets#bucket-create
func (client *Client) CreateBucket(bucket *Bucket) (*Bucket, error) {
	log.Printf("[DEBUG] creating bucket %s", bucket.Name)
	data := url.Values{}
	data.Add("name", bucket.Name)
	data.Add("team_uuid", bucket.Team.ID)

	log.Printf("[DEBUG] 	request: POST %s %#v", "/buckets", data)

	req, err := client.newFormURLEncodedRequest("POST", "/buckets", data)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] %#v", req)
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	log.Printf("[DEBUG] 	response: %d %s", resp.StatusCode, bodyString)

	if resp.StatusCode >= 300 {
		errorResp := new(errorResponse)
		if err = json.Unmarshal(bodyBytes, &errorResp); err != nil {
			return nil, fmt.Errorf("Error creating bucket: %s", bucket.Name)
		}

		return nil, fmt.Errorf("Error creating bucket: %s, status: %d reason: %q", bucket.Name,
			errorResp.Status, errorResp.ErrorMessage)

	}

	response := new(response)
	json.Unmarshal(bodyBytes, &response)
	return getBucketFromResponse(response.Data)
}

// ReadBucket list details about an existing bucket resource. See https://www.runscope.com/docs/api/buckets#bucket-list
func (client *Client) ReadBucket(key string) (*Bucket, error) {
	resource, error := client.readResource("bucket", key, fmt.Sprintf("/buckets/%s", key))
	if error != nil {
		return nil, error
	}

	bucket, error := getBucketFromResponse(resource.Data)
	return bucket, error
}

// DeleteBucket deletes a bucket by key. See https://www.runscope.com/docs/api/buckets#bucket-delete
func (client *Client) DeleteBucket(key string) error {
	return client.deleteResource("bucket", key, fmt.Sprintf("/buckets/%s", key))
}

func (bucket *Bucket) String() string {
	value, err := json.Marshal(bucket)
	if err != nil {
		return ""
	}

	return string(value)
}

func getBucketFromResponse(response interface{}) (*Bucket, error) {
	bucket := new(Bucket)
	err := decode(bucket, response)
	return bucket, err
}
