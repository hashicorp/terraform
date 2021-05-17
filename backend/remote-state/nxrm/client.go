package nxrm

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform/states/remote"
	"github.com/hashicorp/terraform/states/statemgr"
)

type RemoteClient struct {
	userName       string
	password       string
	url            string
	subpath        string
	stateName      string
	timeout        int
	tfLockArtifact string
	httpClient     *http.Client

	lockID       string
	jsonLockInfo []byte
}

func (n *RemoteClient) getNXRMURL(artifact string) string {
	url := n.url
	if strings.HasSuffix(n.url, "/") {
		url = strings.TrimRight(n.url, "/")
	}

	subpath := n.subpath
	if strings.HasSuffix(n.subpath, "/") {
		subpath = strings.TrimRight(n.subpath, "/")
	}

	if strings.HasPrefix(n.subpath, "/") {
		subpath = strings.TrimLeft(n.subpath, "/")
	}

	return fmt.Sprintf("%s/%s/%s", url, subpath, artifact)
}

func (n *RemoteClient) getHTTPClient() *http.Client {
	if n.httpClient == nil {
		n.httpClient = &http.Client{
			Timeout: time.Second * time.Duration(n.timeout),
		}
	}
	return n.httpClient
}

func (n *RemoteClient) getRequest(method string, artifact string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, n.getNXRMURL(artifact), data)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(n.userName, n.password)

	return req, nil
}

func (n *RemoteClient) Get() (*remote.Payload, error) {
	req, err := n.getRequest(http.MethodGet, n.stateName, nil)
	if err != nil {
		return nil, err
	}

	res, err := n.getHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, nil
	}

	output, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	hash := md5.Sum(output)

	payload := &remote.Payload{
		Data: output,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (n *RemoteClient) Put(data []byte) error {
	req, err := n.getRequest(http.MethodPut, n.stateName, bytes.NewReader(data))
	if err != nil {
		return err
	}

	_, err = n.getHTTPClient().Do(req)
	if err != nil {
		return err
	}

	return nil
}

func (n *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	jsonLockInfo := info.Marshal()

	req, err := n.getRequest(http.MethodGet, n.tfLockArtifact, nil)
	if err != nil {
		return "", err
	}

	resp, err := n.getHTTPClient().Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		js := make(map[string]interface{})
		err = json.Unmarshal(body, &js)
		if err != nil {
			return "", err
		}

		id := js["ID"].(string)

		return "", fmt.Errorf("NXRM remote state already locked: ID=%s", id)
	}

	if resp.StatusCode == http.StatusNotFound {
		req, err := n.getRequest(http.MethodPut, n.tfLockArtifact, bytes.NewReader(jsonLockInfo))
		if err != nil {
			return "", err
		}

		resp, err := n.getHTTPClient().Do(req)
		if err != nil {
			return "", err
		}

		switch resp.StatusCode {
		case http.StatusCreated:
			n.lockID = info.ID
			n.jsonLockInfo = jsonLockInfo
			return info.ID, nil
		case http.StatusUnauthorized:
			return "", fmt.Errorf("NXRM requires auth")
		case http.StatusForbidden:
			return "", fmt.Errorf("NXRM invalid auth")
		case http.StatusBadRequest:
			return info.ID, nil
		case http.StatusConflict, http.StatusLocked:
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("NXRM remote state already locked, failed to read body")
			}
			existing := statemgr.LockInfo{}
			err = json.Unmarshal(body, &existing)
			if err != nil {
				return "", fmt.Errorf("NXRM remote state already locked, failed to unmarshal body")
			}

		default:
			return "", fmt.Errorf("unexpected HTTP response code %d", resp.StatusCode)
		}
	}

	return "", fmt.Errorf("unexpected HTTP response code %d", resp.StatusCode)
}

func (n *RemoteClient) Unlock(id string) error {
	// @TODO Sanity check this! Not sure it should exist
	lockErr := &statemgr.LockError{}
	if n.lockID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}

	req, err := n.getRequest(http.MethodGet, n.tfLockArtifact, nil)
	if err != nil {
		return err
	}

	resp, err := n.getHTTPClient().Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		req, err := n.getRequest(http.MethodDelete, n.tfLockArtifact, nil)
		if err != nil {
			return err
		}

		resp, err := n.getHTTPClient().Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}
	}

	return fmt.Errorf("unexpected HTTP response code %d", resp.StatusCode)
}

func (n *RemoteClient) Delete() error {
	req, err := n.getRequest(http.MethodDelete, n.stateName, nil)
	if err != nil {
		return err
	}

	_, err = n.getHTTPClient().Do(req)
	if err != nil {
		return err
	}

	return nil
}
