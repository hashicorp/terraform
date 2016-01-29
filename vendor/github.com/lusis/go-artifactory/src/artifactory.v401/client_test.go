package artifactory

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	_ "io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	_ "os"
	"testing"
)

func TestNewClientCustomTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "pong")
	}))
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}
	defer server.Close()
	conf := &ClientConfig{
		BaseURL:   "http://127.0.0.1:8080/",
		Username:  "username",
		Password:  "password",
		VerifySSL: false,
		Transport: transport,
	}

	client := NewClient(conf)
	res, err := client.Get("/ping", make(map[string]string))
	assert.Nil(t, err, "should not return an error")
	assert.Equal(t, "pong", string(res), "should return the testmsg")
}
