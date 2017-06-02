package gitlab

import (
	"fmt"
	"net/url"
)

// BuildVariablesService handles communication with the project variables related methods
// of the Gitlab API
//
// Gitlab API Docs : https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md
type BuildVariablesService struct {
	client *Client
}

// BuildVariable represents a variable available for each build of the given project
//
// Gitlab API Docs : https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md
type BuildVariable struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (v BuildVariable) String() string {
	return Stringify(v)
}

// ListBuildVariablesOptions are the parameters to ListBuildVariables()
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#list-project-variables
type ListBuildVariablesOptions struct {
	ListOptions
}

// ListBuildVariables gets the a list of project variables in a project
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#list-project-variables
func (s *BuildVariablesService) ListBuildVariables(pid interface{}, opts *ListBuildVariablesOptions, options ...OptionFunc) ([]*BuildVariable, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/variables", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opts, options)
	if err != nil {
		return nil, nil, err
	}

	var v []*BuildVariable
	resp, err := s.client.Do(req, &v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// GetBuildVariable gets a single project variable of a project
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#show-variable-details
func (s *BuildVariablesService) GetBuildVariable(pid interface{}, key string, options ...OptionFunc) (*BuildVariable, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/variables/%s", url.QueryEscape(project), key)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	v := new(BuildVariable)
	resp, err := s.client.Do(req, v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// CreateBuildVariable creates a variable for a given project
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#create-variable
func (s *BuildVariablesService) CreateBuildVariable(pid interface{}, key, value string, options ...OptionFunc) (*BuildVariable, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/variables", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, BuildVariable{key, value}, options)
	if err != nil {
		return nil, nil, err
	}

	v := new(BuildVariable)
	resp, err := s.client.Do(req, v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// UpdateBuildVariable updates an existing project variable
// The variable key must exist
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#update-variable
func (s *BuildVariablesService) UpdateBuildVariable(pid interface{}, key, value string, options ...OptionFunc) (*BuildVariable, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/variables/%s", url.QueryEscape(project), key)

	req, err := s.client.NewRequest("PUT", u, BuildVariable{key, value}, options)
	if err != nil {
		return nil, nil, err
	}

	v := new(BuildVariable)
	resp, err := s.client.Do(req, v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// RemoveBuildVariable removes a project variable of a given project identified by its key
//
// Gitlab API Docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_variables.md#remove-variable
func (s *BuildVariablesService) RemoveBuildVariable(pid interface{}, key string, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/variables/%s", url.QueryEscape(project), key)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
