package winrm

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/masterzen/winrm/soap"
	. "gopkg.in/check.v1"
)

func (s *WinRMSuite) TestExecuteCommand(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}
	count := 0
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		switch count {
		case 0:
			{
				c.Assert(message.String(), Contains, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
				count = 1
				return executeCommandResponse, nil
			}
		case 1:
			{
				c.Assert(message.String(), Contains, "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive")
				count = 2
				return outputResponse, nil
			}
		default:
			{
				return doneCommandResponse, nil
			}
		}
	}

	command, _ := shell.Execute("ipconfig /all")
	var stdout, stderr bytes.Buffer
	var wg sync.WaitGroup
	f := func(b *bytes.Buffer, r *commandReader) {
		wg.Add(1)
		defer wg.Done()
		io.Copy(b, r)
	}
	go f(&stdout, command.Stdout)
	go f(&stderr, command.Stderr)
	command.Wait()
	wg.Wait()
	c.Assert(stdout.String(), Equals, "That's all folks!!!")
	c.Assert(stderr.String(), Equals, "This is stderr, I'm pretty sure!")
}

func (s *WinRMSuite) TestStdinCommand(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}
	count := 0
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		if strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Send") {
			c.Assert(message.String(), Contains, "c3RhbmRhcmQgaW5wdXQ=")
			return "", nil
		} else {
			if strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command") {
				return executeCommandResponse, nil
			} else if count != 1 && strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive") {
				count = 1
				return outputResponse, nil
			} else {
				return doneCommandResponse, nil
			}
		}
	}

	command, _ := shell.Execute("ipconfig /all")
	command.Stdin.Write([]byte("standard input"))
	// slurp output from command
	var outWriter, errWriter bytes.Buffer
	go io.Copy(&outWriter, command.Stdout)
	go io.Copy(&errWriter, command.Stderr)
	command.Wait()
}

func (s *WinRMSuite) TestCommandExitCode(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}
	count := 0
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		defer func() { count += 1 }()
		switch count {
		case 0:
			return executeCommandResponse, nil
		case 1:
			return doneCommandResponse, nil
		default:
			c.Log("Mimicking some observed Windows behavior where only the first 'done' response has the actual exit code and 0 afterwards")
			return doneCommandExitCode0Response, nil
		}
	}

	command, _ := shell.Execute("ipconfig /all")

	command.Wait()
	<-time.After(time.Second) // to make the test fail if fetchOutput races to re-set the exit code

	c.Assert(command.ExitCode(), Equals, 123)
}

func (s *WinRMSuite) TestCloseCommandStopsFetch(c *C) {
	client, err := NewClient(&Endpoint{Host: "localhost", Port: 5985}, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	shell := &Shell{client: client, ShellId: "67A74734-DD32-4F10-89DE-49A060483810"}

	http := make(chan string)
	client.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		switch {
		case strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive"):
			c.Log("Request for command output received by server")
			r := <-http
			c.Log("Returning command output")
			return r, nil
		case strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command"):
			return executeCommandResponse, nil
		case strings.Contains(message.String(), "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal"):
			c.Log("Signal message received by server")
			return "", nil // response is not used
		default:
			c.Logf("Unexpected message: %s", message)
			return "", nil
		}
	}

	command, _ := shell.Execute("ipconfig /all")
	// need to be reading Stdout/Stderr, otherwise, the writes to these are blocking...
	go ioutil.ReadAll(command.Stdout)
	go ioutil.ReadAll(command.Stderr)

	http <- outputResponse // wait for command to enter fetch/slurp

	command.Close()

	select {
	case http <- outputResponse: // return to fetch from slurp
		c.Log("Fetch loop 'drained' one last reponse before realizing that the command is now closed")
	case <-time.After(1 * time.Second):
		c.Log("no poll within one second, fetch may have stopped")
	}

	select {
	case http <- outputResponse:
		c.Log("Fetch loop is still polling after command.Close()")
		c.FailNow()
	case <-time.After(1 * time.Second):
		c.Log("no poll within one second, assuming fetch has stopped")
	}
}
