package command

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

const remoteStateResponseText = `
{
	"version": 1,
	"serial": 1,
	"remote": {
		"type": "http",
		"config": {
			"address": "http://127.0.0.1:12345/",
			"skip_cert_verification": "0"
		}
	},
	"modules": [{
		"path": [
			"root"
		],
		"outputs": {
			"foo": "bar",
			"baz": "qux"
		},
		"resources": {}
	},{
		"path": [
			"root",
			"my_module"
		],
		"outputs": {
			"blah": "tastatur",
			"baz": "qux"
		},
		"resources": {}
	}]
}
`

const remoteStateResponseTextNoState = `
{
	"version": 0,
	"serial": 0,
	"remote": {
		"type": "http",
		"config": {
			"address": "http://127.0.0.1:12345/",
			"skip_cert_verification": "0"
		}
	},
	"modules": []
}
`

const remoteStateResponseTextNoVars = `
{
	"version": 1,
	"serial": 1,
	"remote": {
		"type": "http",
		"config": {
			"address": "http://127.0.0.1:12345/",
			"skip_cert_verification": "0"
		}
	},
	"modules": [{
		"path": [
			"root"
		],
		"outputs": {},
		"resources": {}
	}
}
`

// newRemoteStateHTTPTestServer retuns a HTTP test server.
func newRemoteStateHTTPTestServer(f func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(f))
	return ts
}

// httpRemoteStateTestServer returns a fully configured HTTP test server for
// HTTP remote state.
func httpRemoteStateTestServer(response string) *httptest.Server {
	return newRemoteStateHTTPTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		http.Error(w, response, http.StatusOK)
	})
}

// runTestRemoteOutputRequest is a helper function that performs the common
// tasks of setting up the HTTP test server and sending the
// "terraform remote output" command, and returns the exit code and output.
func runTestRemoteOutputRequest(extraArgs []string, response string) (string, int) {
	ts := httpRemoteStateTestServer(response)
	defer ts.Close()

	ui := new(cli.MockUi)
	c := &RemoteOutputCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		"-backend=http",
		"-backend-config=address=" + ts.URL,
	}

	for _, v := range extraArgs {
		args = append(args, v)
	}

	code := c.Run(args)
	out := ui.OutputWriter.String()

	return out, code
}

func TestRemoteOutput(t *testing.T) {
	text, code := runTestRemoteOutputRequest([]string{}, remoteStateResponseText)

	if code != 0 {
		t.Fatalf("bad: \n%s", text)
	}

	// Our output needs to be sorted here, remote state returns unordered.
	expectedOutput := strings.Split("\n", "foo = bar\nbaz = qux\n")
	sort.Strings(expectedOutput)

	output := strings.Split("\n", text)
	sort.Strings(output)
	if reflect.DeepEqual(output, expectedOutput) != true {
		t.Fatalf("Expected output: %#v\ngiven: %#v", expectedOutput, output)
	}
}

func TestRemoteOutput_moduleSingle(t *testing.T) {
	args := []string{
		"-module", "my_module",
		"blah",
	}

	text, code := runTestRemoteOutputRequest(args, remoteStateResponseText)

	if code != 0 {
		t.Fatalf("bad: \n%s", text)
	}

	actual := strings.TrimSpace(text)
	if actual != "tastatur" {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestRemoteOutput_moduleAll(t *testing.T) {

	args := []string{
		"-module", "my_module",
		"",
	}

	text, code := runTestRemoteOutputRequest(args, remoteStateResponseText)

	if code != 0 {
		t.Fatalf("bad: \n%s", text)
	}

	expectedOutput := strings.Split("\n", "blah = tastatur\nbaz = qux\n")
	sort.Strings(expectedOutput)

	output := strings.Split("\n", text)
	sort.Strings(output)
	if reflect.DeepEqual(output, expectedOutput) != true {
		t.Fatalf("Expected output: %#v\ngiven: %#v", expectedOutput, output)
	}
}

func TestRemoteOutput_missingModule(t *testing.T) {
	args := []string{
		"-module", "not_existing_module",
		"blah",
	}

	if text, code := runTestRemoteOutputRequest(args, remoteStateResponseText); code != 1 {
		t.Fatalf("bad: \n%s", text)
	}
}

func TestRemoteOutput_badVar(t *testing.T) {
	args := []string{
		"bar",
	}

	if text, code := runTestRemoteOutputRequest(args, remoteStateResponseText); code != 1 {
		t.Fatalf("bad: \n%s", text)
	}
}

func TestRemoteOutput_manyArgs(t *testing.T) {
	args := []string{
		"bad",
		"bad",
	}

	if text, code := runTestRemoteOutputRequest(args, remoteStateResponseText); code != 1 {
		t.Fatalf("bad: \n%s", text)
	}
}

func TestRemoteOutput_noArgs(t *testing.T) {
	ui := new(cli.MockUi)
	c := &RemoteOutputCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}

func TestRemoteOutput_noState(t *testing.T) {
	args := []string{
		"foo",
	}
	if text, code := runTestRemoteOutputRequest(args, remoteStateResponseTextNoState); code != 1 {
		t.Fatalf("bad: \n%s", text)
	}
}

func TestRemoteOutput_noVars(t *testing.T) {
	args := []string{
		"bar",
	}
	if text, code := runTestRemoteOutputRequest(args, remoteStateResponseTextNoVars); code != 1 {
		t.Fatalf("bad: \n%s", text)
	}
}
