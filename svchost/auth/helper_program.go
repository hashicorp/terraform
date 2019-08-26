package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"

	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/svchost"
)

type helperProgramCredentialsSource struct {
	executable string
	args       []string
}

// HelperProgramCredentialsSource returns a CredentialsSource that runs the
// given program with the given arguments in order to obtain credentials.
//
// The given executable path must be an absolute path; it is the caller's
// responsibility to validate and process a relative path or other input
// provided by an end-user. If the given path is not absolute, this
// function will panic.
//
// When credentials are requested, the program will be run in a child process
// with the given arguments along with two additional arguments added to the
// end of the list: the literal string "get", followed by the requested
// hostname in ASCII compatibility form (punycode form).
func HelperProgramCredentialsSource(executable string, args ...string) CredentialsSource {
	if !filepath.IsAbs(executable) {
		panic("NewCredentialsSourceHelperProgram requires absolute path to executable")
	}

	fullArgs := make([]string, len(args)+1)
	fullArgs[0] = executable
	copy(fullArgs[1:], args)

	return &helperProgramCredentialsSource{
		executable: executable,
		args:       fullArgs,
	}
}

func (s *helperProgramCredentialsSource) ForHost(host svchost.Hostname) (HostCredentials, error) {
	args := make([]string, len(s.args), len(s.args)+2)
	copy(args, s.args)
	args = append(args, "get")
	args = append(args, string(host))

	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}

	cmd := exec.Cmd{
		Path:   s.executable,
		Args:   args,
		Stdin:  nil,
		Stdout: &outBuf,
		Stderr: &errBuf,
	}
	err := cmd.Run()
	if _, isExitErr := err.(*exec.ExitError); isExitErr {
		errText := errBuf.String()
		if errText == "" {
			// Shouldn't happen for a well-behaved helper program
			return nil, fmt.Errorf("error in %s, but it produced no error message", s.executable)
		}
		return nil, fmt.Errorf("error in %s: %s", s.executable, errText)
	} else if err != nil {
		return nil, fmt.Errorf("failed to run %s: %s", s.executable, err)
	}

	var m map[string]interface{}
	err = json.Unmarshal(outBuf.Bytes(), &m)
	if err != nil {
		return nil, fmt.Errorf("malformed output from %s: %s", s.executable, err)
	}

	return HostCredentialsFromMap(m), nil
}

func (s *helperProgramCredentialsSource) StoreForHost(host svchost.Hostname, credentials HostCredentialsWritable) error {
	args := make([]string, len(s.args), len(s.args)+2)
	copy(args, s.args)
	args = append(args, "store")
	args = append(args, string(host))

	toStore := credentials.ToStore()
	toStoreRaw, err := ctyjson.Marshal(toStore, toStore.Type())
	if err != nil {
		return fmt.Errorf("can't serialize credentials to store: %s", err)
	}

	inReader := bytes.NewReader(toStoreRaw)
	errBuf := bytes.Buffer{}

	cmd := exec.Cmd{
		Path:   s.executable,
		Args:   args,
		Stdin:  inReader,
		Stderr: &errBuf,
		Stdout: nil,
	}
	err = cmd.Run()
	if _, isExitErr := err.(*exec.ExitError); isExitErr {
		errText := errBuf.String()
		if errText == "" {
			// Shouldn't happen for a well-behaved helper program
			return fmt.Errorf("error in %s, but it produced no error message", s.executable)
		}
		return fmt.Errorf("error in %s: %s", s.executable, errText)
	} else if err != nil {
		return fmt.Errorf("failed to run %s: %s", s.executable, err)
	}

	return nil
}

func (s *helperProgramCredentialsSource) ForgetForHost(host svchost.Hostname) error {
	args := make([]string, len(s.args), len(s.args)+2)
	copy(args, s.args)
	args = append(args, "forget")
	args = append(args, string(host))

	errBuf := bytes.Buffer{}

	cmd := exec.Cmd{
		Path:   s.executable,
		Args:   args,
		Stdin:  nil,
		Stderr: &errBuf,
		Stdout: nil,
	}
	err := cmd.Run()
	if _, isExitErr := err.(*exec.ExitError); isExitErr {
		errText := errBuf.String()
		if errText == "" {
			// Shouldn't happen for a well-behaved helper program
			return fmt.Errorf("error in %s, but it produced no error message", s.executable)
		}
		return fmt.Errorf("error in %s: %s", s.executable, errText)
	} else if err != nil {
		return fmt.Errorf("failed to run %s: %s", s.executable, err)
	}

	return nil
}
