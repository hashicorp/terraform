package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type capture struct {
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (c capture) ToJson(structured bool) (interface{}, error) {
	var target []byte
	if structured {
		list := strings.Split(c.stdout.String(), "\n")
		var filtered []string
		for _, part := range list {
			if len(part) > 0 {
				filtered = append(filtered, part)
			}
		}
		target = []byte(fmt.Sprintf("[%s]", strings.Join(filtered, ",")))
	} else {
		target = c.stdout.Bytes()
	}

	var data interface{}
	if err := json.Unmarshal(target, &data); err != nil {
		return data, err
	}
	return data, nil
}

func (c capture) ToError() error {
	str := c.stderr.String()
	if len(str) > 0 {
		return errors.New(str)
	}
	return nil
}

func Capture(cmd *exec.Cmd) *capture {
	out := capture{
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	cmd.Stdout = out.stdout
	cmd.Stderr = out.stderr
	return &out
}
