//
// Copyright 2016, Arkbriar
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
	"bytes"
	"fmt"
	"io"
	"net/url"
	"time"
)

// ListBuildsOptions are options for two list apis
type ListBuildsOptions struct {
	ListOptions
	Scope []BuildState `url:"scope,omitempty" json:"scope,omitempty"`
}

// BuildsService handles communication with the ci builds related methods
// of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md
type BuildsService struct {
	client *Client
}

// Build represents a ci build.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md
type Build struct {
	Commit        *Commit    `json:"commit"`
	CreatedAt     *time.Time `json:"created_at"`
	ArtifactsFile struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	} `json:"artifacts_file"`
	FinishedAt *time.Time `json:"finished_at"`
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Ref        string     `json:"ref"`
	Runner     struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
		IsShared    bool   `json:"is_shared"`
		Name        string `json:"name"`
	} `json:"runner"`
	Stage     string     `json:"stage"`
	StartedAt *time.Time `json:"started_at"`
	Status    string     `json:"status"`
	Tag       bool       `json:"tag"`
	User      *User      `json:"user"`
}

// ListProjectBuilds gets a list of builds in a project.
//
// The scope of builds to show, one or array of: pending, running,
// failed, success, canceled; showing all builds if none provided.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#list-project-builds
func (s *BuildsService) ListProjectBuilds(pid interface{}, opts *ListBuildsOptions, options ...OptionFunc) ([]Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opts, options)
	if err != nil {
		return nil, nil, err
	}

	var builds []Build
	resp, err := s.client.Do(req, &builds)
	if err != nil {
		return nil, resp, err
	}

	return builds, resp, err
}

// ListCommitBuilds gets a list of builds for specific commit in a
// project. If the commit SHA is not found, it will respond with 404.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#list-commit-builds
func (s *BuildsService) ListCommitBuilds(pid interface{}, sha string, opts *ListBuildsOptions, options ...OptionFunc) ([]Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s/builds", project, sha)

	req, err := s.client.NewRequest("GET", u, opts, options)
	if err != nil {
		return nil, nil, err
	}

	var builds []Build
	resp, err := s.client.Do(req, &builds)
	if err != nil {
		return nil, resp, err
	}

	return builds, resp, err
}

// GetBuild gets a single build of a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#get-a-single-build
func (s *BuildsService) GetBuild(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d", project, buildID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}

// GetBuildArtifacts get builds artifacts of a project
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#get-build-artifacts
func (s *BuildsService) GetBuildArtifacts(pid interface{}, buildID int, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/artifacts", project, buildID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	artifactsBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, artifactsBuf)
	if err != nil {
		return nil, resp, err
	}

	return artifactsBuf, resp, err
}

// DownloadArtifactsFile download the artifacts file from the given
// reference name and job provided the build finished successfully.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#download-the-artifacts-file
func (s *BuildsService) DownloadArtifactsFile(pid interface{}, refName string, job string, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/artifacts/%s/download?job=%s", project, refName, job)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	artifactsBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, artifactsBuf)
	if err != nil {
		return nil, resp, err
	}

	return artifactsBuf, resp, err
}

// GetTraceFile gets a trace of a specific build of a project
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#get-a-trace-file
func (s *BuildsService) GetTraceFile(pid interface{}, buildID int, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/trace", project, buildID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	traceBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, traceBuf)
	if err != nil {
		return nil, resp, err
	}

	return traceBuf, resp, err
}

// CancelBuild cancels a single build of a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#cancel-a-build
func (s *BuildsService) CancelBuild(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/cancel", project, buildID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}

// RetryBuild retries a single build of a project
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#retry-a-build
func (s *BuildsService) RetryBuild(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/retry", project, buildID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}

// EraseBuild erases a single build of a project, removes a build
// artifacts and a build trace.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#erase-a-build
func (s *BuildsService) EraseBuild(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/erase", project, buildID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}

// KeepArtifacts prevents artifacts from being deleted when
// expiration is set.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#keep-artifacts
func (s *BuildsService) KeepArtifacts(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/artifacts/keep", project, buildID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}

// PlayBuild triggers a nanual action to start a build.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/builds.md#play-a-build
func (s *BuildsService) PlayBuild(pid interface{}, buildID int, options ...OptionFunc) (*Build, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/builds/%d/play", project, buildID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	build := new(Build)
	resp, err := s.client.Do(req, build)
	if err != nil {
		return nil, resp, err
	}

	return build, resp, err
}
