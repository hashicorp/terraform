// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oras

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

// GHCR workarounds - they return 405 for OCI DELETE, so we hit their REST API instead.
// TODO(future): check if GHCR ever fixes this, it's been years...

var errNotGHCR = errors.New("not a ghcr.io repository")

const (
	ghcrHost              = "ghcr.io"
	githubAPIBaseURL      = "https://api.github.com"
	githubAPIVersion      = "2022-11-28"
	githubMaxVersionPages = 20
	githubVersionsPerPage = 100
)

type githubPackageVersion struct {
	ID       int64 `json:"id"`
	Metadata struct {
		Container struct {
			Tags []string `json:"tags"`
		} `json:"container"`
	} `json:"metadata"`
}

// tryDeleteGHCRTag deletes via GitHub Packages API when OCI DELETE fails.
func tryDeleteGHCRTag(ctx context.Context, repo *orasRepositoryClient, tag string) error {
	if repo == nil {
		return fmt.Errorf("nil repository client")
	}

	host, owner, packageName, err := parseGHCRRepository(repo.repository)
	if err != nil {
		return err
	}
	if host != ghcrHost {
		return errNotGHCR
	}

	token, err := repo.accessTokenForHost(ctx, host)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("no credentials available for %s (need a token with delete:packages)", host)
	}

	return deleteGitHubPackageVersionByTag(ctx, githubAPIBaseURL, owner, packageName, tag, token)
}

func parseGHCRRepository(repository string) (host, owner, packageName string, err error) {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid repository %q: expected <host>/<owner>/<name>", repository)
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/"), nil
}

// deleteGitHubPackageVersionByTag tries orgs endpoint first, then users.
func deleteGitHubPackageVersionByTag(ctx context.Context, baseURL, owner, packageName, tag, token string) error {
	baseURL = strings.TrimRight(baseURL, "/")
	pkgEscaped := url.PathEscape(packageName)
	ownerEscaped := url.PathEscape(owner)

	// Try orgs endpoint first
	orgBase := fmt.Sprintf("%s/orgs/%s/packages/container/%s", baseURL, ownerEscaped, pkgEscaped)
	if err := deleteFromGitHubPackagesEndpoint(ctx, orgBase, tag, token); err == nil {
		return nil
	} else if !isHTTPStatus(err, http.StatusNotFound) {
		return err // Return non-404 errors (permission issues, etc.)
	}

	// Fall back to users endpoint
	userBase := fmt.Sprintf("%s/users/%s/packages/container/%s", baseURL, ownerEscaped, pkgEscaped)
	return deleteFromGitHubPackagesEndpoint(ctx, userBase, tag, token)
}

func deleteFromGitHubPackagesEndpoint(ctx context.Context, baseURL, tag, token string) error {
	client := cleanhttp.DefaultClient()

	versionID, err := findGitHubVersionIDByTag(ctx, client, baseURL, tag, token)
	if err != nil {
		return err
	}
	if versionID == 0 {
		return newHTTPStatusError(http.StatusNotFound, "tag not found in package versions")
	}

	deleteURL := fmt.Sprintf("%s/versions/%d", baseURL, versionID)
	return githubRequest(ctx, client, http.MethodDelete, deleteURL, token, "delete package version", http.StatusNoContent, nil)
}

func findGitHubVersionIDByTag(ctx context.Context, client *http.Client, baseURL, tag, token string) (int64, error) {
	for page := 1; page <= githubMaxVersionPages; page++ {
		versions, err := listGitHubPackageVersions(ctx, client, baseURL, page, token)
		if err != nil {
			return 0, err
		}
		if len(versions) == 0 {
			break
		}

		if id := findVersionIDWithTag(versions, tag); id != 0 {
			return id, nil
		}
	}
	return 0, nil
}

func listGitHubPackageVersions(ctx context.Context, client *http.Client, baseURL string, page int, token string) ([]githubPackageVersion, error) {
	versionsURL, err := url.Parse(baseURL + "/versions")
	if err != nil {
		return nil, err
	}
	q := versionsURL.Query()
	q.Set("per_page", fmt.Sprintf("%d", githubVersionsPerPage))
	q.Set("page", fmt.Sprintf("%d", page))
	versionsURL.RawQuery = q.Encode()

	var versions []githubPackageVersion
	if err := githubRequest(ctx, client, http.MethodGet, versionsURL.String(), token, "list package versions", http.StatusOK, &versions); err != nil {
		return nil, err
	}
	return versions, nil
}

func findVersionIDWithTag(versions []githubPackageVersion, tag string) int64 {
	for _, v := range versions {
		for _, t := range v.Metadata.Container.Tags {
			if t == tag {
				return v.ID
			}
		}
	}
	return 0
}

func githubRequest(ctx context.Context, client *http.Client, method, urlStr, token, operation string, expectedStatus int, decode any) error {
	req, err := http.NewRequestWithContext(ctx, method, urlStr, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return newHTTPStatusError(resp.StatusCode, operation)
	}
	if decode == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(decode)
}

type httpStatusErr struct {
	code int
	op   string
}

func (e httpStatusErr) Error() string {
	return fmt.Sprintf("github api error (%s): status %d", e.op, e.code)
}

func newHTTPStatusError(code int, op string) error {
	return httpStatusErr{code: code, op: op}
}

func isHTTPStatus(err error, code int) bool {
	var e httpStatusErr
	if errors.As(err, &e) {
		return e.code == code
	}
	return false
}
