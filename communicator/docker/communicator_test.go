package docker

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"regexp"
	"strings"

	"log"

	"archive/tar"

	"io"

	"bytes"

	"fmt"

	"math/rand"
	"strconv"

	"reflect"

	"encoding/json"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

var url string

type singleFileUploadHandler struct {
	fileName string
	content  string
}

func (s *singleFileUploadHandler) dockerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("Docker API mock: Got a request %v %v", r.Method, r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "_ping") {
		w.WriteHeader(http.StatusOK)
	} else if matched, _ := regexp.MatchString("/containers/[[:alnum:]]+/archive$", r.URL.Path); matched && r.Method == "PUT" {
		reader := tar.NewReader(r.Body)
		var header *tar.Header
		var err error
		if header, err = reader.Next(); err != nil {
			log.Printf("Docker API mock: tr.Next: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if header.Name != s.fileName {
			log.Printf("Docker API mock: File name doesn't match: expected %q but actually got %q", "terraform.file", header.Name)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var b bytes.Buffer
		if _, err := io.Copy(&b, reader); err != nil {
			log.Printf("Docker API mock: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if b.String() != s.content {
			log.Printf("Docker API mock: File content doesn't match: expected %q but actually got %q", "terraform.file", header.Name)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		log.Printf("Docker API mock: method not supported (501)")
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func TestCommunicator_FileUpload(t *testing.T) {
	expectedFileName := "terraform.file"
	expectedContent := "Hello world!"
	sfuh := &singleFileUploadHandler{
		fileName: expectedFileName,
		content:  expectedContent,
	}
	ts := httptest.NewServer(http.HandlerFunc(sfuh.dockerHandler))
	defer ts.Close()
	expectedHost := ts.URL
	expectedContainerId := "045c63979a"
	s := &terraform.InstanceState{
		ID: expectedContainerId,
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "docker",
				"host": expectedHost,
			},
		},
	}
	var comm *Communicator
	var err error
	if comm, err = New(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	if err = comm.Connect(nil); err != nil {
		t.Fatalf("Error at connect %v", err)
	}

	if err = comm.Upload(fmt.Sprintf("/tmp/%s", expectedFileName), strings.NewReader(expectedContent)); err != nil {
		t.Fatalf("Upload error %v", err)
	}

}

func TestCommunicator_ScriptUpload(t *testing.T) {
	expectedFileName := "terraform.sh"
	inlineScript := "echo Hello world!"
	expectedContent := fmt.Sprintf("#!/bin/sh\n%s", inlineScript)
	sfuh := &singleFileUploadHandler{
		fileName: expectedFileName,
		content:  expectedContent,
	}
	ts := httptest.NewServer(http.HandlerFunc(sfuh.dockerHandler))
	defer ts.Close()
	expectedHost := ts.URL
	expectedContainerId := "045c63979a"
	s := &terraform.InstanceState{
		ID: expectedContainerId,
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type": "docker",
				"host": expectedHost,
			},
		},
	}
	var comm *Communicator
	var err error
	if comm, err = New(s); err != nil {
		t.Fatalf("err: %v", err)
	}

	if err = comm.Connect(nil); err != nil {
		t.Fatalf("Error at connect %v", err)
	}

	if err = comm.UploadScript(fmt.Sprintf("/tmp/%s", expectedFileName), strings.NewReader(inlineScript)); err != nil {
		t.Fatalf("Upload Script error %v", err)
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
		comm := &Communicator{connInfo: &connectionInfo{ScriptPath: tc.Input}}
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

type execCommandHandler struct {
	expectedCommand []string
	execResult      int
	execID          string
}

type execRequest struct {
	Cmd []string `json:"Cmd,omitempty"`
}

func (s *execCommandHandler) dockerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("Docker API mock: Got a request %v %v", r.Method, r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "_ping") {
		w.WriteHeader(http.StatusOK)
	} else if matched, _ := regexp.MatchString("/containers/[[:alnum:]]+/archive$", r.URL.Path); matched && r.Method == "PUT" {
		w.WriteHeader(http.StatusOK)
	} else if matched, _ := regexp.MatchString("/containers/[[:alnum:]]+/exec", r.URL.Path); matched && r.Method == "POST" {
		var payload execRequest
		body, _ := ioutil.ReadAll(r.Body)
		log.Printf("%s", string(body))
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Docker API mock: Unmarshaling json failed %v", err)
			return
		}
		if !reflect.DeepEqual(payload.Cmd, s.expectedCommand) {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Docker API mock: Unexpected Exec cmd %+v expected %+v", payload.Cmd, s.expectedCommand)
			return
		}
		w.WriteHeader(http.StatusOK)
		s.execID = strconv.FormatInt(int64(rand.Int31()), 10)
		w.Write([]byte(fmt.Sprintf("{\"Id\": \"%s\",\"Warnings\":[]}", s.execID)))
	} else if matched, _ := regexp.MatchString(fmt.Sprintf("/exec/%s/json", s.execID), r.URL.Path); matched && r.Method == "GET" {
		if len(s.execID) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Docker API mock: Exec details called before creating an Exec")
			return
		}
		w.WriteHeader(http.StatusOK)
		s.execID = strconv.FormatInt(int64(rand.Int31()), 10)
		w.Write([]byte(fmt.Sprintf(`{
  "ID" : "045c63979a",
  "Running" : false,
  "ExitCode" : %d,
  "ProcessConfig" : {
    "privileged" : false,
    "user" : "",
    "tty" : false,
    "entrypoint" : "echo",
    "arguments" : [
      "hello",
      "world!"
    ]
  },
  "OpenStdin" : false,
  "OpenStderr" : false,
  "OpenStdout" : false
}`, s.execResult)))
	} else if matched, _ := regexp.MatchString(fmt.Sprintf("/exec/%s/start", s.execID), r.URL.Path); matched && r.Method == "POST" {
		if len(s.execID) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Docker API mock: Exec Start called before creating an Exec")
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		log.Printf("Docker API mock: method not supported (501)")
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func TestCommunicator_Start(t *testing.T) {
	cases := []struct {
		expectedCommand    []string
		expectedExecResult int
	}{
		{
			[]string{"echo", "Hello", "world!"},
			0,
		},
		{
			[]string{"exit", "1"},
			1,
		},
	}

	for _, tc := range cases {
		ech := &execCommandHandler{
			expectedCommand: tc.expectedCommand,
			execResult:      tc.expectedExecResult,
		}
		ts := httptest.NewServer(http.HandlerFunc(ech.dockerHandler))
		defer ts.Close()
		expectedHost := ts.URL
		expectedContainerId := "045c63979a"
		s := &terraform.InstanceState{
			ID: expectedContainerId,
			Ephemeral: terraform.EphemeralState{
				ConnInfo: map[string]string{
					"type": "docker",
					"host": expectedHost,
				},
			},
		}
		var comm *Communicator
		var err error
		if comm, err = New(s); err != nil {
			t.Fatalf("err: %v", err)
		}

		if err = comm.Connect(nil); err != nil {
			t.Fatalf("Error at connect %v", err)
		}

		cmd := &remote.Cmd{
			Command: strings.Join(tc.expectedCommand, "   "),
		}
		if err = comm.Start(cmd); err != nil {
			t.Fatalf("Upload error %v", err)
		}

		if cmd.ExitStatus != tc.expectedExecResult {
			t.Fatalf("Command Exit Status expected: %d but actually got %d", tc.expectedExecResult, cmd.ExitStatus)
		}
	}
}
