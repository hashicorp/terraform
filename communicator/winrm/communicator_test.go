package winrm

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"testing"

	"github.com/dylanmei/winrmtest"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

func newMockWinRMServer(t *testing.T) *winrmtest.Remote {
	wrm := winrmtest.NewRemote()

	wrm.CommandFunc(
		winrmtest.MatchText("echo foo"),
		func(out, err io.Writer) int {
			out.Write([]byte("foo"))
			return 0
		})

	wrm.CommandFunc(
		winrmtest.MatchPattern(`^echo c29tZXRoaW5n >> ".*"$`),
		func(out, err io.Writer) int {
			return 0
		})

	wrm.CommandFunc(
		winrmtest.MatchPattern(`^powershell.exe -EncodedCommand .*$`),
		func(out, err io.Writer) int {
			return 0
		})

	wrm.CommandFunc(
		winrmtest.MatchText("powershell"),
		func(out, err io.Writer) int {
			return 0
		})

	return wrm
}

func TestStart(t *testing.T) {
	wrm := newMockWinRMServer(t)
	defer wrm.Close()

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "winrm",
				"user":     "user",
				"password": "pass",
				"host":     wrm.Host,
				"port":     strconv.Itoa(wrm.Port),
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	err = c.Start(&cmd)
	if err != nil {
		t.Fatalf("error executing remote command: %s", err)
	}
	cmd.Wait()

	if stdout.String() != "foo" {
		t.Fatalf("bad command response: expected %q, got %q", "foo", stdout.String())
	}
}

func TestUpload(t *testing.T) {
	wrm := newMockWinRMServer(t)
	defer wrm.Close()

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "winrm",
				"user":     "user",
				"password": "pass",
				"host":     wrm.Host,
				"port":     strconv.Itoa(wrm.Port),
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	err = c.Connect(nil)
	if err != nil {
		t.Fatalf("error connecting communicator: %s", err)
	}
	defer c.Disconnect()

	err = c.Upload("C:/Temp/terraform.cmd", bytes.NewReader([]byte("something")))
	if err != nil {
		t.Fatalf("error uploading file: %s", err)
	}
}

func TestScriptPath(t *testing.T) {
	cases := []struct {
		Input   string
		Pattern string
	}{
		{
			"/tmp/script.sh",
			`^/tmp/script\.sh$`,
		},
		{
			"/tmp/script_%RAND%.sh",
			`^/tmp/script_(\d+)\.sh$`,
		},
	}

	for _, tc := range cases {
		r := &terraform.InstanceState{
			Ephemeral: terraform.EphemeralState{
				ConnInfo: map[string]string{
					"type":        "winrm",
					"script_path": tc.Input,
				},
			},
		}
		comm, err := New(r)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		output := comm.ScriptPath()

		match, err := regexp.Match(tc.Pattern, []byte(output))
		if err != nil {
			t.Fatalf("bad: %s\n\nerr: %s", tc.Input, err)
		}
		if !match {
			t.Fatalf("bad: %s\n\n%s", tc.Input, output)
		}
	}
}

func TestNoTransportDecorator(t *testing.T) {
	wrm := newMockWinRMServer(t)
	defer wrm.Close()

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "winrm",
				"user":     "user",
				"password": "pass",
				"host":     wrm.Host,
				"port":     strconv.Itoa(wrm.Port),
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	err = c.Connect(nil)
	if err != nil {
		t.Fatalf("error connecting communicator: %s", err)
	}
	defer c.Disconnect()

	if c.client.TransportDecorator != nil {
		t.Fatal("bad TransportDecorator: expected nil, got non-nil")
	}
}

func TestTransportDecorator(t *testing.T) {
	wrm := newMockWinRMServer(t)
	defer wrm.Close()

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "winrm",
				"user":     "user",
				"password": "pass",
				"host":     wrm.Host,
				"port":     strconv.Itoa(wrm.Port),
				"use_ntlm": "true",
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	err = c.Connect(nil)
	if err != nil {
		t.Fatalf("error connecting communicator: %s", err)
	}
	defer c.Disconnect()

	if c.client.TransportDecorator == nil {
		t.Fatal("bad TransportDecorator: expected non-nil, got nil")
	}
}

func TestScriptPath_randSeed(t *testing.T) {
	// Pre GH-4186 fix, this value was the deterministic start the pseudorandom
	// chain of unseeded math/rand values for Int31().
	staticSeedPath := "C:/Temp/terraform_1298498081.cmd"
	c, err := New(&terraform.InstanceState{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	path := c.ScriptPath()
	if path == staticSeedPath {
		t.Fatalf("rand not seeded! got: %s", path)
	}
}
