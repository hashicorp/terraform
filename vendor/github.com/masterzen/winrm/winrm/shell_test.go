package winrm

import (
	"github.com/masterzen/winrm/soap"
	. "gopkg.in/check.v1"
)

func (s *WinRMSuite) TestShellExecuteResponse(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}
	first := true
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		if first {
			c.Assert(message.String(), Contains, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
			first = false
			return executeCommandResponse, nil
		} else {
			c.Assert(message.String(), Contains, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive")
			return outputResponse, nil
		}
	}

	command, _ := shell.Execute("ipconfig /all")
	c.Assert(command.commandId, Equals, "1A6DEE6B-EC68-4DD6-87E9-030C0048ECC4")
}

func (s *WinRMSuite) TestShellCloseResponse(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		c.Assert(message.String(), Contains, "http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete")
		return "", nil
	}

	shell.Close()
}
