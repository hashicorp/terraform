package winrm

import (
	"strings"
	"testing"

	"github.com/masterzen/simplexml/dom"
	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type WinRMSuite struct{}

var _ = Suite(&WinRMSuite{})

func (s *WinRMSuite) TestOpenShellRequest(c *C) {
	openShell := NewOpenShellRequest("http://localhost", nil)
	defer openShell.Free()

	assertXPath(c, openShell.Doc(), "//a:Action", "http://schemas.xmlsoap.org/ws/2004/09/transfer/Create")
	assertXPath(c, openShell.Doc(), "//a:To", "http://localhost")
	assertXPath(c, openShell.Doc(), "//env:Body/rsp:Shell/rsp:InputStreams", "stdin")
	assertXPath(c, openShell.Doc(), "//env:Body/rsp:Shell/rsp:OutputStreams", "stdout stderr")
}

func (s *WinRMSuite) TestDeleteShellRequest(c *C) {
	request := NewDeleteShellRequest("http://localhost", "SHELLID", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
}

func (s *WinRMSuite) TestExecuteCommandRequest(c *C) {
	request := NewExecuteCommandRequest("http://localhost", "SHELLID", "ipconfig /all", []string{}, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//w:Option[@Name=\"WINRS_CONSOLEMODE_STDIN\"]", "TRUE")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Command", "ipconfig /all")
	assertXPathNil(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments")
}

func (s *WinRMSuite) TestExecuteCommandWithArgumentsRequest(c *C) {
	args := []string{"/p", "C:\\test.txt"}
	request := NewExecuteCommandRequest("http://localhost", "SHELLID", "del", args, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//w:Option[@Name=\"WINRS_CONSOLEMODE_STDIN\"]", "TRUE")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Command", "del")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments", "/p")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments", "C:\\test.txt")
}

func (s *WinRMSuite) TestGetOutputRequest(c *C) {
	request := NewGetOutputRequest("http://localhost", "SHELLID", "COMMANDID", "stdout stderr", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Receive/rsp:DesiredStream[@CommandId=\"COMMANDID\"]", "stdout stderr")
}

func (s *WinRMSuite) TestSendInputRequest(c *C) {
	request := NewSendInputRequest("http://localhost", "SHELLID", "COMMANDID", []byte{31, 32}, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Send")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Send/rsp:Stream[@CommandId=\"COMMANDID\"]", "HyA=")
}

func (s *WinRMSuite) TestSignalRequest(c *C) {
	request := NewSignalRequest("http://localhost", "SHELLID", "COMMANDID", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Signal[@CommandId=\"COMMANDID\"]/rsp:Code", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/signal/terminate")
}

func assertXPath(c *C, doc *dom.Document, request string, expected string) {
	root, path, err := parseXPath(doc, request)

	if err != nil {
		c.Fatalf("Xpath %s gives error %s", request, err)
	}

	ok := path.Exists(root)
	c.Assert(ok, Equals, true)

	var foundValue string
	iter := path.Iter(root)
	for iter.Next() {
		foundValue = iter.Node().String()
		if foundValue == expected {
			break
		}
	}

	if foundValue != expected {
		c.Errorf("Should have found '%s', but found '%s' instead", expected, foundValue)
	}
}

func assertXPathNil(c *C, doc *dom.Document, request string) {
	root, path, err := parseXPath(doc, request)

	if err != nil {
		c.Fatalf("Xpath %s gives error %s", request, err)
	}

	ok := path.Exists(root)
	c.Assert(ok, Equals, false)
}

func parseXPath(doc *dom.Document, request string) (*xmlpath.Node, *xmlpath.Path, error) {
	content := strings.NewReader(doc.String())
	node, err := xmlpath.Parse(content)
	if err != nil {
		return nil, nil, err
	}

	path, err := xmlpath.CompileWithNamespace(request, soap.GetAllNamespaces())
	if err != nil {
		return nil, nil, err
	}

	return node, path, nil
}
