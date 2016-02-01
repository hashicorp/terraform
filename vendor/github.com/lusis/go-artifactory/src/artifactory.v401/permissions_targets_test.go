package artifactory

import (
	_ "bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestGetPermissions(t *testing.T) {
	responseFile, err := os.Open("assets/test/permissions.json")
	if err != nil {
		t.Fatalf("Unable to read test data: %s", err.Error())
	}
	defer responseFile.Close()
	responseBody, _ := ioutil.ReadAll(responseFile)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, string(responseBody))
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	conf := &ClientConfig{
		BaseURL:   "http://127.0.0.1:8080/",
		Username:  "username",
		Password:  "password",
		VerifySSL: false,
		Transport: transport,
	}

	client := NewClient(conf)
	perms, err := client.GetPermissionTargets()
	assert.NoError(t, err, "should not return an error")
	assert.Len(t, perms, 5, "should have five targets")
	assert.Equal(t, perms[0].Name, "snapshot-write", "Should have the snapshot-write target")
	assert.Equal(t, perms[0].Uri, "https://artifactory/artifactory/api/security/permissions/snapshot-write", "should have a uri")
	for _, p := range perms {
		assert.NotNil(t, p.Name, "Name should not be empty")
		assert.NotNil(t, p.Uri, "Uri should not be empty")
	}
}

func TestGetPermissionDetails(t *testing.T) {
	responseFile, err := os.Open("assets/test/permissions_details.json")
	if err != nil {
		t.Fatalf("Unable to read test data: %s", err.Error())
	}
	defer responseFile.Close()
	responseBody, _ := ioutil.ReadAll(responseFile)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, string(responseBody))
	}))
	defer server.Close()

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	conf := &ClientConfig{
		BaseURL:   "http://127.0.0.1:8080/",
		Username:  "username",
		Password:  "password",
		VerifySSL: false,
		Transport: transport,
	}

	client := NewClient(conf)
	perms, err := client.GetPermissionTargetDetails("release-commiter")
	assert.NoError(t, err, "should not return an error")
	assert.Equal(t, perms.Name, "release-commiter", "Should be release-commiter")
	assert.Equal(t, perms.IncludesPattern, "**", "Includes should be **")
	assert.Equal(t, perms.ExcludesPattern, "", "Excludes should be nil")
	assert.Len(t, perms.Repositories, 3, "Should have 3 repositories")
	assert.Contains(t, perms.Repositories, "docker-local-v2", "Should have repos")
	assert.NotNil(t, perms.Principals.Groups, "should have a group principal")
	groups := []string{}
	for g := range perms.Principals.Groups {
		groups = append(groups, g)
	}
	assert.Contains(t, groups, "java-committers", "Should have the java committers group")
	assert.Len(t, perms.Principals.Groups["java-committers"], 4, "should have 4 permissions")
	assert.Contains(t, perms.Principals.Groups["java-committers"], "r", "Should have the r permission")
}
