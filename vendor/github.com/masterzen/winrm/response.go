package winrm

import (
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
)

func first(node *xmlpath.Node, xpath string) (string, error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return "", err
	}
	content, _ := path.String(node)
	return content, nil
}

func any(node *xmlpath.Node, xpath string) (bool, error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return false, err
	}

	return path.Exists(node), nil

}

func xpath(node *xmlpath.Node, xpath string) ([]xmlpath.Node, error) {
	path, err := xmlpath.CompileWithNamespace(xpath, soap.GetAllNamespaces())
	if err != nil {
		return nil, err
	}

	nodes := make([]xmlpath.Node, 0, 1)
	iter := path.Iter(node)
	for iter.Next() {
		nodes = append(nodes, *(iter.Node()))
	}

	return nodes, nil
}

func ParseOpenShellResponse(response string) (string, error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))
	if err != nil {
		return "", err
	}
	return first(doc, "//w:Selector[@Name='ShellId']")
}

func ParseExecuteCommandResponse(response string) (string, error) {
	doc, err := xmlpath.Parse(strings.NewReader(response))
	if err != nil {
		return "", err
	}
	return first(doc, "//rsp:CommandId")
}

func ParseSlurpOutputErrResponse(response string, stdout, stderr io.Writer) (bool, int, error) {
	var (
		finished bool
		exitCode int
	)

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

	return finished, exitCode, err
}

func ParseSlurpOutputResponse(response string, stream io.Writer, streamType string) (bool, int, error) {
	var (
		finished bool
		exitCode int
	)

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

	return finished, exitCode, err
}
