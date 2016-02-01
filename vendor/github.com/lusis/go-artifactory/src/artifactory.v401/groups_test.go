package artifactory

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestGetGroups(t *testing.T) {
	responseFile, err := os.Open("assets/test/groups.json")
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
	groups, err := client.GetGroups()
	assert.NoError(t, err, "should not return an error")
	assert.Len(t, groups, 5, "should have five groups")
	assert.Equal(t, groups[0].Name, "administrators", "Should have the administrators group")
	assert.Equal(t, groups[0].Uri, "https://artifactory/artifactory/api/security/groups/administrators", "should have a uri")
	for _, g := range groups {
		assert.NotNil(t, g.Name, "Name should not be empty")
		assert.NotNil(t, g.Uri, "Uri should not be empty")
	}
}

func TestGetGroupDetails(t *testing.T) {
	responseFile, err := os.Open("assets/test/single_group.json")
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
	group, err := client.GetGroupDetails("docker-readers")
	assert.NoError(t, err, "should not return an error")
	assert.Equal(t, group.Name, "docker-readers", "name should be docker-readers")
	assert.Equal(t, group.Description, "Can read from Docker repositories", "description should match")
	assert.False(t, group.AutoJoin, "autojoin should be false")
	assert.Equal(t, group.Realm, "artifactory", "realm should be artifactory")
}

func TestCreateGroupNoDetails(t *testing.T) {
	var buf bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		req, _ := ioutil.ReadAll(r.Body)
		buf.Write(req)
		fmt.Fprintf(w, "")
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
	var details GroupDetails = GroupDetails{}
	err := client.CreateGroup("testgroup", details)
	assert.NoError(t, err, "should not return an error")
	assert.Equal(t, "{}", string(buf.Bytes()), "should send empty json")
}

func TestCreateGroupDetails(t *testing.T) {
	var buf bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		req, _ := ioutil.ReadAll(r.Body)
		buf.Write(req)
		fmt.Fprintf(w, "")
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
	var details GroupDetails = GroupDetails{
		Description: "test group desc",
		AutoJoin:    true,
	}
	expectedJson := `{"description":"test group desc","autoJoin":true}`
	err := client.CreateGroup("testgroup", details)
	assert.NoError(t, err, "should not return an error")
	assert.Equal(t, expectedJson, string(buf.Bytes()), "should send empty json")
}
