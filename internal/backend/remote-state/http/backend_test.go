package http

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestHTTPClientFactory(t *testing.T) {
	// defaults

	conf := map[string]cty.Value{
		"address": cty.StringVal("http://127.0.0.1:8888/foo"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, address")
	}
	if client.URL.String() != "http://127.0.0.1:8888/foo" {
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
	conf = map[string]cty.Value{
		"address":        cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":  cty.StringVal("BLAH"),
		"lock_address":   cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":    cty.StringVal("BLIP"),
		"unlock_address": cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":  cty.StringVal("BLOOP"),
		"username":       cty.StringVal("user"),
		"password":       cty.StringVal("pass"),
		"retry_max":      cty.StringVal("999"),
		"retry_wait_min": cty.StringVal("15"),
		"retry_wait_max": cty.StringVal("150"),
	}

	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	if client == nil {
		t.Fatal("Unexpected failure, update_method")
	}
	if client.UpdateMethod != "BLAH" {
		t.Fatalf("Expected update_method \"%s\", got \"%s\"", "BLAH", client.UpdateMethod)
	}
	if client.LockURL.String() != conf["lock_address"].AsString() || client.LockMethod != "BLIP" {
		t.Fatalf("Unexpected lock_address \"%s\" vs \"%s\" or lock_method \"%s\" vs \"%s\"", client.LockURL.String(),
			conf["lock_address"].AsString(), client.LockMethod, conf["lock_method"])
	}
	if client.UnlockURL.String() != conf["unlock_address"].AsString() || client.UnlockMethod != "BLOOP" {
		t.Fatalf("Unexpected unlock_address \"%s\" vs \"%s\" or unlock_method \"%s\" vs \"%s\"", client.UnlockURL.String(),
			conf["unlock_address"].AsString(), client.UnlockMethod, conf["unlock_method"])
	}
	if client.Username != "user" || client.Password != "pass" {
		t.Fatalf("Unexpected username \"%s\" vs \"%s\" or password \"%s\" vs \"%s\"", client.Username, conf["username"],
			client.Password, conf["password"])
	}
	if client.Client.RetryMax != 999 {
		t.Fatalf("Expected retry_max \"%d\", got \"%d\"", 999, client.Client.RetryMax)
	}
	if client.Client.RetryWaitMin != 15*time.Second {
		t.Fatalf("Expected retry_wait_min \"%s\", got \"%s\"", 15*time.Second, client.Client.RetryWaitMin)
	}
	if client.Client.RetryWaitMax != 150*time.Second {
		t.Fatalf("Expected retry_wait_max \"%s\", got \"%s\"", 150*time.Second, client.Client.RetryWaitMax)
	}
}

func TestHTTPClientFactoryWithEnv(t *testing.T) {
	// env
	conf := map[string]string{
		"address":        "http://127.0.0.1:8888/foo",
		"update_method":  "BLAH",
		"lock_address":   "http://127.0.0.1:8888/bar",
		"lock_method":    "BLIP",
		"unlock_address": "http://127.0.0.1:8888/baz",
		"unlock_method":  "BLOOP",
		"username":       "user",
		"password":       "pass",
		"retry_max":      "999",
		"retry_wait_min": "15",
		"retry_wait_max": "150",
	}

	defer testWithEnv(t, "TF_HTTP_ADDRESS", conf["address"])()
	defer testWithEnv(t, "TF_HTTP_UPDATE_METHOD", conf["update_method"])()
	defer testWithEnv(t, "TF_HTTP_LOCK_ADDRESS", conf["lock_address"])()
	defer testWithEnv(t, "TF_HTTP_UNLOCK_ADDRESS", conf["unlock_address"])()
	defer testWithEnv(t, "TF_HTTP_LOCK_METHOD", conf["lock_method"])()
	defer testWithEnv(t, "TF_HTTP_UNLOCK_METHOD", conf["unlock_method"])()
	defer testWithEnv(t, "TF_HTTP_USERNAME", conf["username"])()
	defer testWithEnv(t, "TF_HTTP_PASSWORD", conf["password"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_MAX", conf["retry_max"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_WAIT_MIN", conf["retry_wait_min"])()
	defer testWithEnv(t, "TF_HTTP_RETRY_WAIT_MAX", conf["retry_wait_max"])()

	b := backend.TestBackendConfig(t, New(), nil).(*Backend)
	client := b.client

	if client == nil {
		t.Fatal("Unexpected failure, EnvDefaultFunc")
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
	if client.Client.RetryMax != 999 {
		t.Fatalf("Expected retry_max \"%d\", got \"%d\"", 999, client.Client.RetryMax)
	}
	if client.Client.RetryWaitMin != 15*time.Second {
		t.Fatalf("Expected retry_wait_min \"%s\", got \"%s\"", 15*time.Second, client.Client.RetryWaitMin)
	}
	if client.Client.RetryWaitMax != 150*time.Second {
		t.Fatalf("Expected retry_wait_max \"%s\", got \"%s\"", 150*time.Second, client.Client.RetryWaitMax)
	}
}

// testWithEnv sets an environment variable and returns a deferable func to clean up
func testWithEnv(t *testing.T, key string, value string) func() {
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("err: %v", err)
	}

	return func() {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}
