package http

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/states/remote"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestHTTPWorkspaceUrlFunction(t *testing.T) {
	conf := map[string]cty.Value{
		"address": cty.StringVal("http://127.0.0.1:8888/foo"),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)

	// no path
	orig := "http://127.0.0.1"
	u, _ := url.Parse(orig)
	u2, err := b.workspaceUrlSubstitute(u, "test", "tset")
	if err != nil {
		t.Fatal("unexpected error from url substitute function")
	}
	if u2.String() != orig {
		t.Fatal("string returned has been changed")
	}

	// simple path
	orig = "http://127.0.0.1/element/teststring/else"
	u, _ = url.Parse(orig)
	u2, err = b.workspaceUrlSubstitute(u, "test", "tset")
	if err != nil {
		t.Fatal("unexpected error from url substitute function 2")
	}
	if u2.String() != "http://127.0.0.1/element/tsetstring/else" {
		t.Fatal("string returned not mutated correctly")
	}

	// escaped path
	orig = "http://127.0.0.1/element/<teststring>/else"
	u, _ = url.Parse(orig)
	u2, err = b.workspaceUrlSubstitute(u, "<teststring>", ">tset<")
	if err != nil {
		t.Fatal("unexpected error from url substitute function 3")
	}
	if u2.String() != "http://127.0.0.1/element/%3Etset%3C/else" {
		t.Fatal("string returned not mutated correctly with escaped path")
	}

	// multi replace
	orig = "http://127.0.0.1/element/teststringtest/else"
	u, _ = url.Parse(orig)
	u2, err = b.workspaceUrlSubstitute(u, "test", "tset")
	if err != nil {
		t.Fatal("unexpected error from url substitute function 4")
	}
	if u2.String() != "http://127.0.0.1/element/tsetstringtset/else" {
		t.Fatal("string returned not mutated correctly in multi replace")
	}
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

	// workspace disabled
	conf = map[string]cty.Value{
		"address":                  cty.StringVal("http://127.0.0.1:8888/foo"),
		"workspace_path_element":   cty.StringVal("cheese"),
		"workspace_list_address":   cty.StringVal("http://127.0.0.1:8888/workspace/list"),
		"workspace_delete_address": cty.StringVal("http://127.0.0.1:8888/workspace/cheese/delete"),
	}
	b = backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)
	client = b.client

	// ensure with workspaces disabled, all of this is unset
	if client.WorkspaceListURL != nil {
		t.Fatal("unexpected value set in WorkspaceListURL")
	}
	if client.WorkspaceListMethod != "" {
		t.Fatal("unexpected value set in WorkspaceListMethod")
	}
	if client.WorkspaceDeleteURL != nil {
		t.Fatal("unexpected value set in WorkspaceDeleteURL")
	}
	if client.WorkspaceDeleteMethod != "" {
		t.Fatal("unexpected value set in WorkspaceDeleteMethod")
	}
	if b.workspacePathElement != "" {
		t.Fatal("unexpected value set in workspacePathElement")
	}

	// check state manager works for default
	_, err := b.StateMgr("default")
	if err != nil {
		t.Fatal("unexpected error getting default state manager")
	}

	// check workspace functions return expected errors
	_, err = b.StateMgr("test")
	if err != ErrWorkspaceDisabled {
		t.Fatal("incorrect error response for a workspace state when workspaces disabled")
	}

	// workspace list should fail
	_, err = b.Workspaces()
	if err != ErrWorkspaceDisabled {
		t.Fatal("incorrect error response for workspace list when workspaces disabled")
	}

	// workspace delete should fail
	err = b.DeleteWorkspace("test")
	if err != ErrWorkspaceDisabled {
		t.Fatal("incorrect error response for workspace delete when workspaces disabled")
	}

	// custom
	conf = map[string]cty.Value{
		"address":                  cty.StringVal("http://127.0.0.1:8888/foo"),
		"update_method":            cty.StringVal("BLAH"),
		"lock_address":             cty.StringVal("http://127.0.0.1:8888/bar"),
		"lock_method":              cty.StringVal("BLIP"),
		"unlock_address":           cty.StringVal("http://127.0.0.1:8888/baz"),
		"unlock_method":            cty.StringVal("BLOOP"),
		"username":                 cty.StringVal("user"),
		"password":                 cty.StringVal("pass"),
		"retry_max":                cty.StringVal("999"),
		"retry_wait_min":           cty.StringVal("15"),
		"retry_wait_max":           cty.StringVal("150"),
		"workspace_enabled":        cty.BoolVal(true),
		"workspace_path_element":   cty.StringVal("cheese"),
		"workspace_list_address":   cty.StringVal("http://127.0.0.1:8888/workspace/list"),
		"workspace_delete_address": cty.StringVal("http://127.0.0.1:8888/workspace/cheese/delete"),
		"headers": cty.ObjectVal(map[string]cty.Value{
			"X-TOKEN": cty.StringVal("secret"),
		}),
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
	if b.workspaceEnabled != true {
		t.Fatalf("Expected workspace_enabled to be \"%s\" got \"%t\"", conf["workspace_enabled"].AsString(),
			b.workspaceEnabled)
	}
	if b.workspacePathElement != "cheese" {
		t.Fatalf("Expected workspace_path_element \"%s\", got \"%s\"", conf["workspace_path_element"].AsString(),
			b.workspacePathElement)
	}
	if client.WorkspaceListURL.String() != conf["workspace_list_address"].AsString() {
		t.Fatalf("Unexpected workspace_list_url \"%s\", got \"%s\"", conf["workspace_list_address"].AsString(),
			client.WorkspaceListURL.String())
	}
	if client.WorkspaceDeleteURL.String() != conf["workspace_delete_address"].AsString() {
		t.Fatalf("Unexpected workspace_delete_url \"%s\", got \"%s\"", conf["workspace_delete_address"].AsString(),
			client.WorkspaceDeleteURL.String())
	}
	if client.Headers == nil {
		t.Fatalf("Unexpected nil map for client headers")
	}
	if client.Headers["X-TOKEN"] != "secret" {
		t.Fatalf("Unexpected headers entry \"%s\", got \"%s\"", "secret",
			client.Headers["X-TOKEN"])
	}
}

func TestWithClient(t *testing.T) {
	handler, ts, u1, err := createTestServer()
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	defer ts.Close()

	handler.Data["/list"] = "[\"workspace1\",\"workspace2\"]"

	u2, err := u1.Parse("/state/REPLACE")
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	u3, err := u1.Parse("/list")
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}

	conf := map[string]cty.Value{
		"address":                cty.StringVal(u2.String()),
		"lock_address":           cty.StringVal(u2.String() + "/lock"),
		"unlock_address":         cty.StringVal(u2.String() + "/unlock"),
		"workspace_enabled":      cty.BoolVal(true),
		"workspace_path_element": cty.StringVal("REPLACE"),
		"workspace_list_address": cty.StringVal(u3.String()),
	}
	b := backend.TestBackendConfig(t, New(), configs.SynthBody("synth", conf)).(*Backend)

	str, err := b.Workspaces()
	if err != nil {
		t.Fatal("unexpected error from Workspaces()")
	}
	if len(str) != 2 && str[0] != "workspace1" && str[1] != "workspace2" {
		t.Fatalf("unexpected workspace list contents, got %+v", str)
	}

	testState := func(wsname string) {
		stateMgr, err := b.StateMgr(wsname)
		if err != nil {
			t.Fatal("unexpected error when getting state manager")
		}

		ut, err := u1.Parse("/state/" + wsname)
		if err != nil {
			t.Fatalf("Parse: %s", err)
		}
		rs, ok := stateMgr.(*remote.State)
		if !ok {
			t.Fatal("failed to assert statemgr type")
		}
		rc, ok := rs.Client.(*httpClient)
		if !ok {
			t.Fatal("failed to assert httpclient type")
		}

		if rc.URL.String() != ut.String() {
			t.Fatalf("state url incorrect, got %s, expected %s", rc.URL.String(), ut.String())
		}

		lock := ut.String() + "/lock"
		unlock := ut.String() + "/unlock"
		if rc.LockURL.String() != lock {
			t.Fatalf("lock url incorrect, got %s, expected %s", rc.LockURL.String(), lock)
		}
		if rc.UnlockURL.String() != unlock {
			t.Fatalf("unlock url incorrect, got %s, expected %s", rc.UnlockURL.String(), unlock)
		}

		handler.Data["/state/"+wsname] = "{}"
		err = b.DeleteWorkspace("notthis")
		if _, ok := handler.Data["/state/"+wsname]; !ok {
			t.Fatal("unexpected unrelated workspace deleted")
		}
		err = b.DeleteWorkspace(wsname)
		if _, ok := handler.Data["/state/"+wsname]; ok {
			t.Fatal("workspace not deleted")
		}
	}

	testState("workspace1")
	// check we can safely create multiple statemgrs
	testState("workspace2")
}

func TestHTTPClientFactoryWithEnv(t *testing.T) {
	// env
	conf := map[string]string{
		"address":                  "http://127.0.0.1:8888/foo",
		"update_method":            "BLAH",
		"lock_address":             "http://127.0.0.1:8888/bar",
		"lock_method":              "BLIP",
		"unlock_address":           "http://127.0.0.1:8888/baz",
		"unlock_method":            "BLOOP",
		"username":                 "user",
		"password":                 "pass",
		"retry_max":                "999",
		"retry_wait_min":           "15",
		"retry_wait_max":           "150",
		"skip_cert":                "true",
		"workspace_path_element":   "cheese",
		"workspace_list_address":   "http://127.0.0.1:8888/ws/list",
		"workspace_list_method":    "PUT",
		"workspace_delete_address": "http://127.0.0.1:8888/ws/del/cheese",
		"workspace_delete_method":  "GET",
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
	defer testWithEnv(t, "TF_HTTP_SKIP_CERT", conf["skip_cert"])()
	defer testWithEnv(t, "TF_HTTP_HEADERS", "{\"X-TEST\":\"bob\"}")()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_ENABLED", "true")()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_PATH_ELEMENT", conf["workspace_path_element"])()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_LIST_ADDRESS", conf["workspace_list_address"])()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_LIST_METHOD", conf["workspace_list_method"])()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_DELETE_ADDRESS", conf["workspace_delete_address"])()
	defer testWithEnv(t, "TF_HTTP_WORKSPACE_DELETE_METHOD", conf["workspace_delete_method"])()

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
	if !client.Client.HTTPClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Fatal("Expected skip_cert \"true\", got \"false\"")
	}
	if v, ok := client.Headers["X-TEST"]; !ok || v != "bob" {
		t.Fatalf("Expected http header X-TEST, got %+v", client.Headers)
	}
	if !b.workspaceEnabled {
		t.Fatal("expected workspace to be enabled, it is not")
	}
	if b.workspacePathElement != conf["workspace_path_element"] {
		t.Fatalf("expected workspace path element to equal \"%s\" got \"%s\"", conf["workspace_path_element"], b.workspacePathElement)
	}
	if client.WorkspaceListURL.String() != conf["workspace_list_address"] {
		t.Fatalf("expected workspace_list_url to equal \"%s\" got \"%s\"", conf["workspace_list_address"], client.WorkspaceListURL.String())
	}
	if client.WorkspaceListMethod != conf["workspace_list_method"] {
		t.Fatalf("expected workspace_list_method to equal \"%s\" got \"%s\"", conf["workspace_list_method"], client.WorkspaceListMethod)
	}
	if client.WorkspaceDeleteURL.String() != conf["workspace_delete_address"] {
		t.Fatalf("expected workspace_delete_url to equal \"%s\" got \"%s\"", conf["workspace_delete_address"], client.WorkspaceDeleteURL.String())
	}
	if client.WorkspaceDeleteMethod != conf["workspace_delete_method"] {
		t.Fatalf("expected workspace_delete_method to equal \"%s\" got \"%s\"", conf["workspace_delete_method"], client.WorkspaceDeleteMethod)
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
