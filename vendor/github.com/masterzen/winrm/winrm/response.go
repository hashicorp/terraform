package winrm

import (
	"encoding/base64"
	"fmt"
	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
	"io"
	"strconv"
	"strings"
)

func first(node *xmlpath.Node, xpath string) (content string, err error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return
	}
	content, _ = path.String(node)
	return
}

func any(node *xmlpath.Node, xpath string) (found bool, err error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return
	}

	found = path.Exists(node)
	return
}

func xpath(node *xmlpath.Node, xpath string) (nodes []xmlpath.Node, err error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return
	}

	nodes = make([]xmlpath.Node, 0, 1)
	iter := path.Iter(node)
	for iter.Next() {
		nodes = append(nodes, *(iter.Node()))
	}
	return
}

func ParseOpenShellResponse(response string) (shellId string, err error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))

	shellId, err = first(doc, "//w:Selector[@Name='ShellId']")
	return
}

func ParseExecuteCommandResponse(response string) (commandId string, err error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))

	commandId, err = first(doc, "//rsp:CommandId")
	return
}

func ParseSlurpOutputErrResponse(response string, stdout io.Writer, stderr io.Writer) (finished bool, exitCode int, err error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))

	stdouts, _ := xpath(doc, "//rsp:Stream[@Name='stdout']")
	for _, node := range stdouts {
		content, _ := base64.StdEncoding.DecodeString(node.String())
		stdout.Write(content)
	}
	stderrs, _ := xpath(doc, "//rsp:Stream[@Name='stderr']")
	for _, node := range stderrs {
		content, _ := base64.StdEncoding.DecodeString(node.String())
		stderr.Write(content)
	}

	ended, _ := any(doc, "//*[@State='http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done']")

	if ended {
		finished = ended
		if exitBool, _ := any(doc, "//rsp:ExitCode"); exitBool {
			exit, _ := first(doc, "//rsp:ExitCode")
			exitCode, _ = strconv.Atoi(exit)
		}
	} else {
		finished = false
	}

	return
}

func ParseSlurpOutputResponse(response string, stream io.Writer, streamType string) (finished bool, exitCode int, err error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))

	nodes, _ := xpath(doc, fmt.Sprintf("//rsp:Stream[@Name='%s']", streamType))
	for _, node := range nodes {
		content, _ := base64.StdEncoding.DecodeString(node.String())
		stream.Write(content)
	}

	ended, _ := any(doc, "//*[@State='http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done']")

	if ended {
		finished = ended
		if exitBool, _ := any(doc, "//rsp:ExitCode"); exitBool {
			exit, _ := first(doc, "//rsp:ExitCode")
			exitCode, _ = strconv.Atoi(exit)
		}
	} else {
		finished = false
	}

	return
}
