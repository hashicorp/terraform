//
// Copyright 2015, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"fmt"
	"net/url"
	"time"
)

// DeployKeysService handles communication with the keys related methods
// of the GitLab API.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/deploy_keys.html
type DeployKeysService struct {
	client *Client
}

// DeployKey represents a GitLab deploy key.
type DeployKey struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

func (k DeployKey) String() string {
	return Stringify(k)
}

// ListDeployKeys gets a list of a project's deploy keys
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/deploy_keys.html#list-deploy-keys
func (s *DeployKeysService) ListDeployKeys(pid interface{}) ([]*DeployKey, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/keys", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var k []*DeployKey
	resp, err := s.client.Do(req, &k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// GetDeployKey gets a single deploy key.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/deploy_keys.html#single-deploy-key
func (s *DeployKeysService) GetDeployKey(
	pid interface{},
	deployKey int) (*DeployKey, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/keys/%d", url.QueryEscape(project), deployKey)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	k := new(DeployKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// AddDeployKeyOptions represents the available ADDDeployKey() options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/deploy_keys.html#add-deploy-key
type AddDeployKeyOptions struct {
	Title *string `url:"title,omitempty" json:"title,omitempty"`
	Key   *string `url:"key,omitempty" json:"key,omitempty"`
}

// AddDeployKey creates a new deploy key for a project. If deploy key already
// exists in another project - it will be joined to project but only if
// original one was is accessible by same user.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/deploy_keys.html#add-deploy-key
func (s *DeployKeysService) AddDeployKey(
	pid interface{},
	opt *AddDeployKeyOptions) (*DeployKey, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/keys", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	k := new(DeployKey)
	resp, err := s.client.Do(req, k)
	if err != nil {
		return nil, resp, err
	}

	return k, resp, err
}

// DeleteDeployKey deletes a deploy key from a project.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/deploy_keys.html#delete-deploy-key
func (s *DeployKeysService) DeleteDeployKey(pid interface{}, deployKey int) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/keys/%d", url.QueryEscape(project), deployKey)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
