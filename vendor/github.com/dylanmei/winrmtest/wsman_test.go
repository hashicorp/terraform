package winrmtest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
	"github.com/satori/go.uuid"
)

func Test_creating_a_shell(t *testing.T) {
	w := &wsman{}

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Create</a:Action>
			</env:Header>
			<env:Body>
				<rsp:Shell>
					<rsp:InputStream>stdin</rsp:InputStream>
					<rsp:OutputStreams>stdout stderr</rsp:OutputStreams>
				</rsp:Shell>
			</env:Body>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	if contentType := res.HeaderMap.Get("Content-Type"); contentType != "application/soap+xml" {
		t.Errorf("Expected ContentType application/soap+xml was %s.\n", contentType)
	}

	env, err := xmlpath.Parse(res.Body)
	if err != nil {
		t.Error("Couldn't compile the SOAP response.")
	}

	xpath, _ := xmlpath.CompileWithNamespace(
		"//rsp:ShellId", soap.GetAllNamespaces())

	if _, found := xpath.String(env); !found {
		t.Error("Expected a Shell identifier.")
	}
}

func Test_executing_a_command(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchText("echo tacos"), func(out, err io.Writer) int {
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/shell/Command</a:Action>
			</env:Header>
			<env:Body>
				<rsp:CommandLine><rsp:Command>"echo tacos"</rsp:Command></rsp:CommandLine>
			</env:Body>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	env, err := xmlpath.Parse(res.Body)
	if err != nil {
		t.Error("Couldn't compile the SOAP response.")
	}

	xpath, _ := xmlpath.CompileWithNamespace(
		"//rsp:CommandId", soap.GetAllNamespaces())

	result, _ := xpath.String(env)
	if result != id {
		t.Errorf("Expected CommandId=%s but was \"%s\"", id, result)
	}
}

func Test_executing_a_regex_command(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchPattern(`echo .* >> C:\file.cmd`), func(out, err io.Writer) int {
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(fmt.Sprintf(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/shell/Command</a:Action>
			</env:Header>
			<env:Body>
				<rsp:CommandLine><rsp:Command>"echo %d >> C:\file.cmd"</rsp:Command></rsp:CommandLine>
			</env:Body>
		</env:Envelope>`, uuid.NewV4().String())))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	env, err := xmlpath.Parse(res.Body)
	if err != nil {
		t.Error("Couldn't compile the SOAP response.")
	}

	xpath, _ := xmlpath.CompileWithNamespace(
		"//rsp:CommandId", soap.GetAllNamespaces())

	result, _ := xpath.String(env)
	if result != id {
		t.Errorf("Expected CommandId=%s but was \"%s\"", id, result)
	}
}

func Test_receiving_command_results(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchText("echo tacos"), func(out, err io.Writer) int {
		out.Write([]byte("tacos"))
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(fmt.Sprintf(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive</a:Action>
			</env:Header>
			<env:Body>
				<rsp:Receive><rsp:DesiredStream CommandId="%s">stdout stderr</rsp:DesiredStream></rsp:Receive>
			</env:Body>
		</env:Envelope>`, id)))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	env, err := xmlpath.Parse(res.Body)
	if err != nil {
		t.Error("Couldn't compile the SOAP response.")
	}

	xpath, _ := xmlpath.CompileWithNamespace("//rsp:ReceiveResponse", soap.GetAllNamespaces())
	iter := xpath.Iter(env)
	if !iter.Next() {
		t.Error("Expected a ReceiveResponse element.")
	}

	xresp := iter.Node()
	xpath, _ = xmlpath.CompileWithNamespace(
		fmt.Sprintf("rsp:Stream[@CommandId='%s']", id), soap.GetAllNamespaces())
	iter = xpath.Iter(xresp)

	if !iter.Next() || !nodeHasAttribute(iter.Node(), "Name", "stdout") || iter.Node().String() != "dGFjb3M=" {
		t.Error("Expected an stdout Stream with the text \"dGFjb3M=\".")
	}

	if !iter.Next() || !nodeHasAttribute(iter.Node(), "Name", "stdout") || !nodeHasAttribute(iter.Node(), "End", "true") {
		t.Error("Expected an stdout Stream with an \"End\" attribute.")
	}

	if !iter.Next() || !nodeHasAttribute(iter.Node(), "Name", "stderr") || !nodeHasAttribute(iter.Node(), "End", "true") {
		t.Error("Expected an stderr Stream with an \"End\" attribute.")
	}

	xpath, _ = xmlpath.CompileWithNamespace(
		"//rsp:CommandState[@State='http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done']",
		soap.GetAllNamespaces())

	if _, found := xpath.String(env); !found {
		t.Error("Expected CommandState=\"Done\"")
	}

	xpath, _ = xmlpath.CompileWithNamespace("//rsp:CommandState/rsp:ExitCode", soap.GetAllNamespaces())
	if code, _ := xpath.String(env); code != "0" {
		t.Errorf("Expected ExitCode=0 but found \"%s\"\n", code)
	}
}

func Test_deleting_a_shell(t *testing.T) {
	w := &wsman{}

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete</a:Action>
			</env:Header>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	if res.Body.Len() != 0 {
		t.Errorf("Expected body to be empty but was \"%v\".", res.Body)
	}
}

func nodeHasAttribute(n *xmlpath.Node, name, value string) bool {
	xpath := xmlpath.MustCompile("attribute::" + name)
	if result, found := xpath.String(n); found {
		return result == value
	}

	return false
}
