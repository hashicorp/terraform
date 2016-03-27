package winrm

import (
	"encoding/base64"

	"github.com/masterzen/winrm/soap"
	"github.com/nu7hatch/gouuid"
)

func genUUID() string {
	uuid, _ := uuid.NewV4()
	return "uuid:" + uuid.String()
}

func defaultHeaders(message *soap.SoapMessage, url string, params *Parameters) (h *soap.SoapHeader) {
	h = message.Header()
	h.To(url).ReplyTo("http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous").MaxEnvelopeSize(params.EnvelopeSize).Id(genUUID()).Locale(params.Locale).Timeout(params.Timeout)
	return
}

func NewOpenShellRequest(uri string, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()
	defaultHeaders(message, uri, params).Action("http://schemas.xmlsoap.org/ws/2004/09/transfer/Create").ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").AddOption(soap.NewHeaderOption("WINRS_NOPROFILE", "FALSE")).AddOption(soap.NewHeaderOption("WINRS_CODEPAGE", "65001")).Build()

	body := message.CreateBodyElement("Shell", soap.NS_WIN_SHELL)
	input := message.CreateElement(body, "InputStreams", soap.NS_WIN_SHELL)
	input.SetContent("stdin")
	output := message.CreateElement(body, "OutputStreams", soap.NS_WIN_SHELL)
	output.SetContent("stdout stderr")
	return
}

func NewDeleteShellRequest(uri string, shellId string, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()
	defaultHeaders(message, uri, params).Action("http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete").ShellId(shellId).ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").Build()

	message.NewBody()

	return
}

func NewExecuteCommandRequest(uri, shellId, command string, arguments []string, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()
	defaultHeaders(message, uri, params).Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command").ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").ShellId(shellId).AddOption(soap.NewHeaderOption("WINRS_CONSOLEMODE_STDIN", "TRUE")).AddOption(soap.NewHeaderOption("WINRS_SKIP_CMD_SHELL", "FALSE")).Build()
	body := message.CreateBodyElement("CommandLine", soap.NS_WIN_SHELL)

	// ensure special characters like & don't mangle the request XML
	command = "<![CDATA[" + command + "]]>"
	commandElement := message.CreateElement(body, "Command", soap.NS_WIN_SHELL)
	commandElement.SetContent(command)

	for _, arg := range arguments {
		arg = "<![CDATA[" + arg + "]]>"
		argumentsElement := message.CreateElement(body, "Arguments", soap.NS_WIN_SHELL)
		argumentsElement.SetContent(arg)
	}

	return
}

func NewGetOutputRequest(uri string, shellId string, commandId string, streams string, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()
	defaultHeaders(message, uri, params).Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive").ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").ShellId(shellId).Build()

	receive := message.CreateBodyElement("Receive", soap.NS_WIN_SHELL)
	desiredStreams := message.CreateElement(receive, "DesiredStream", soap.NS_WIN_SHELL)
	desiredStreams.SetAttr("CommandId", commandId)
	desiredStreams.SetContent(streams)
	return
}

func NewSendInputRequest(uri string, shellId string, commandId string, input []byte, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()

	defaultHeaders(message, uri, params).Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Send").ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").ShellId(shellId).Build()

	content := base64.StdEncoding.EncodeToString(input)

	send := message.CreateBodyElement("Send", soap.NS_WIN_SHELL)
	streams := message.CreateElement(send, "Stream", soap.NS_WIN_SHELL)
	streams.SetAttr("Name", "stdin")
	streams.SetAttr("CommandId", commandId)
	streams.SetContent(content)
	return
}

func NewSignalRequest(uri string, shellId string, commandId string, params *Parameters) (message *soap.SoapMessage) {
	if params == nil {
		params = DefaultParameters()
	}
	message = soap.NewMessage()

	defaultHeaders(message, uri, params).Action("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal").ResourceURI("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/cmd").ShellId(shellId).Build()

	signal := message.CreateBodyElement("Signal", soap.NS_WIN_SHELL)
	signal.SetAttr("CommandId", commandId)
	code := message.CreateElement(signal, "Code", soap.NS_WIN_SHELL)
	code.SetContent("http://schemas.microsoft.com/wbem/wsman/1/windows/shell/signal/terminate")

	return
}
