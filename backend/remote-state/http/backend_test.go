package http

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestHTTPClientFactory(t *testing.T) {
	// defaults

	conf := map[string]interface{}{
		"address": "http://127.0.0.1:8888/foo",
	}
	b := backend.TestBackendConfig(t, New(), conf).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != conf["address"] {
		t.Fatalf("Expected address \"%s\", got \"%s\"", conf["address"], client.URL.String())
	}
	if client.UpdateMethod != "POST" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "POST", client.UpdateMethod)
	}
	if client.LockURL != nil || client.LockMethod != "LOCK" {
		t.Fatal("Unexpected lock_address or lock_method")
	}
	if client.UnlockURL != nil || client.UnlockMethod != "UNLOCK" {
		t.Fatal("Unexpected unlock_address or unlock_method")
	}
	if client.Username != "" || client.Password != "" {
		t.Fatal("Unexpected username or password")
	}

	// custom
	conf = map[string]interface{}{
		"address":        "http://127.0.0.1:8888/foo",
		"update_method":  "BLAH",
		"lock_address":   "http://127.0.0.1:8888/bar",
		"lock_method":    "BLIP",
		"unlock_address": "http://127.0.0.1:8888/baz",
		"unlock_method":  "BLOOP",
		"username":       "user",
		"password":       "pass",
	}

	b = backend.TestBackendConfig(t, New(), conf).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"] || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"], client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"] || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"], client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
}
