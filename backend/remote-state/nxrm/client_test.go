package nxrm

import (
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
)

func TestGetNXRMURL(t *testing.T) {
	cfg := InitTestConfig()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)

	url := b.client.getNXRMURL(b.client.stateName)

	if url != `http://localhost:8081/repository/tf-backend/this/here/demo.tfstate` {
		t.Fatalf("getNXRMURL mismatch: %s", url)
	}
}

func TestGetNXRMURLTrimUrl(t *testing.T) {
	cfg := InitTestConfig()
	cfg["url"] = "http://localhost:8081/repository/tf-backend/"
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)

	got := b.client.getNXRMURL(b.client.stateName)
	if got != `http://localhost:8081/repository/tf-backend/this/here/demo.tfstate` {
		t.Fatalf("getNXRMURL mismatch: %s", got)
	}
}

func TestGetNXRMURLTrimSubpathSuffix(t *testing.T) {
	cfg := InitTestConfig()
	cfg["subpath"] = "this/here/"
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)

	got := b.client.getNXRMURL(b.client.stateName)
	if got != `http://localhost:8081/repository/tf-backend/this/here/demo.tfstate` {
		t.Fatalf("getNXRMURL mismatch: %s", got)
	}
}

func TestGetNXRMURLTrimSubpathPrefix(t *testing.T) {
	cfg := InitTestConfig()
	cfg["subpath"] = "/this/here"
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)

	got := b.client.getNXRMURL(b.client.stateName)
	if got != `http://localhost:8081/repository/tf-backend/this/here/demo.tfstate` {
		t.Fatalf("getNXRMURL mismatch: %s", got)
	}
}

func TestGetHTTPClient(t *testing.T) {
	cfg := InitTestConfig()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)
	expectedTimeout := time.Second * time.Duration(cfg["timeout"].(int))

	got := b.client.getHTTPClient()
	if got.Timeout != expectedTimeout {
		t.Fatalf("getHTTPClient returned strange timeout")
	}
}

func TestGetRequest(t *testing.T) {
	cfg := InitTestConfig()
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(cfg)).(*Backend)

	req, err := b.client.getRequest(http.MethodGet, cfg["stateName"].(string), nil)
	if err != nil {
		t.Fatalf("getRequest error: %s", err)
	}
	u, p, ok := req.BasicAuth()
	if !ok {
		t.Fatalf("req.BasicAuth() not ok!")
	}

	if u != cfg["username"].(string) {
		t.Fatalf("GetRequest() - %s", mismatchError(cfg, "username", u))
	}

	if p != cfg["password"].(string) {
		t.Fatalf("GetRequest() - %s", mismatchError(cfg, "password", p))
	}
}
