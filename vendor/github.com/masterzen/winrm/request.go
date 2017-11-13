package winrm

import (
	"encoding/base64"

	"github.com/masterzen/winrm/soap"
	"github.com/nu7hatch/gouuid"
)

func genUUID() string {
	id, _ := uuid.NewV4()
	return "uuid:" + id.String()
}

func defaultHeaders(message *soap.SoapMessage, url string, params *Parameters) *soap.SoapHeader {
	return message.
		Header().
		To(url).
		ReplyTo("http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous").
		MaxEnvelopeSize(params.EnvelopeSize).
		Id(genUUID()).
		Locale(params.Locale).
		Timeout(params.Timeout)
}

//NewOpenShellRequest makes a new soap request
func NewOpenShellRequest(uri string, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}

	message := soap.NewMessage()
	defaultHeaders(message, uri, params).
		Action("http://schemas.xmlsoap.org/ws/2004/09/transfer/Create").
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		AddOption(soap.NewHeaderOption("WINRS_NOPROFILE", "FALSE")).
		AddOption(soap.NewHeaderOption("WINRS_CODEPAGE", "65001")).
		Build()

	body := message.CreateBodyElement("Shell", soap.DOM_NS_WIN_SHELL)
	input := message.CreateElement(body, "InputStreams", soap.DOM_NS_WIN_SHELL)
	input.SetContent("stdin")
	output := message.CreateElement(body, "OutputStreams", soap.DOM_NS_WIN_SHELL)
	output.SetContent("stdout stderr")

	return message
}

// NewDeleteShellRequest ...
func NewDeleteShellRequest(uri, shellId string, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}
	message := soap.NewMessage()
	defaultHeaders(message, uri, params).
		Action("http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete").
		ShellId(shellId).
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		Build()

	message.NewBody()

	return message
}

// NewExecuteCommandRequest exec command on specific shellID
func NewExecuteCommandRequest(uri, shellId, command string, arguments []string, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}
	message := soap.NewMessage()
	defaultHeaders(message, uri, params).
		Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command").
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		ShellId(shellId).
		AddOption(soap.NewHeaderOption("WINRS_CONSOLEMODE_STDIN", "TRUE")).
		AddOption(soap.NewHeaderOption("WINRS_SKIP_CMD_SHELL", "FALSE")).
		Build()

	body := message.CreateBodyElement("CommandLine", soap.DOM_NS_WIN_SHELL)

	// ensure special characters like & don't mangle the request XML
	command = "<![CDATA[" + command + "]]>"
	commandElement := message.CreateElement(body, "Command", soap.DOM_NS_WIN_SHELL)
	commandElement.SetContent(command)

	for _, arg := range arguments {
		arg = "<![CDATA[" + arg + "]]>"
		argumentsElement := message.CreateElement(body, "Arguments", soap.DOM_NS_WIN_SHELL)
		argumentsElement.SetContent(arg)
	}

	return message
}

func NewGetOutputRequest(uri, shellId, commandId, streams string, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}
	message := soap.NewMessage()
	defaultHeaders(message, uri, params).
		Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive").
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		ShellId(shellId).
		Build()

	receive := message.CreateBodyElement("Receive", soap.DOM_NS_WIN_SHELL)
	desiredStreams := message.CreateElement(receive, "DesiredStream", soap.DOM_NS_WIN_SHELL)
	desiredStreams.SetAttr("CommandId", commandId)
	desiredStreams.SetContent(streams)

	return message
}

func NewSendInputRequest(uri, shellId, commandId string, input []byte, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}
	message := soap.NewMessage()

	defaultHeaders(message, uri, params).
		Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Send").
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		ShellId(shellId).
		Build()

	content := base64.StdEncoding.EncodeToString(input)

	send := message.CreateBodyElement("Send", soap.DOM_NS_WIN_SHELL)
	streams := message.CreateElement(send, "Stream", soap.DOM_NS_WIN_SHELL)
	streams.SetAttr("Name", "stdin")
	streams.SetAttr("CommandId", commandId)
	streams.SetContent(content)
	return message
}

func NewSignalRequest(uri string, shellId string, commandId string, params *Parameters) *soap.SoapMessage {
	if params == nil {
		params = DefaultParameters
	}
	message := soap.NewMessage()

	defaultHeaders(message, uri, params).
		Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal").
		ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").
		ShellId(shellId).
		Build()

	signal := message.CreateBodyElement("Signal", soap.DOM_NS_WIN_SHELL)
	signal.SetAttr("CommandId", commandId)
	code := message.CreateElement(signal, "Code", soap.DOM_NS_WIN_SHELL)
	code.SetContent("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/signal/terminate")

	return message
}
