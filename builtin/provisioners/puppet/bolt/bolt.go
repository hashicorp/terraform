package bolt

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Result struct {
	Items []struct {
		Node   string            `json:"node"`
		Status string            `json:"status"`
		Result map[string]string `json:"result"`
	} `json:"items"`
	NodeCount   int `json:"node_count"`
	ElapsedTime int `json:"elapsed_time"`
}

func runCommand(command string, timeout time.Duration) ([]byte, error) {
	var cmdargs []string

	if runtime.GOOS == "windows" {
		cmdargs = []string{"cmd", "/C"}
	} else {
		cmdargs = []string{"/bin/sh", "-c"}
	}
	cmdargs = append(cmdargs, command)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdargs[0], cmdargs[1:]...)
	return cmd.Output()
}

func Task(connInfo map[string]string, timeout time.Duration, sudo bool, task string, args map[string]string) (*Result, error) {
	cmdargs := []string{
		"bolt", "task", "run", "--nodes", connInfo["type"] + "://" + connInfo["host"], "-u", connInfo["user"],
	}

	if connInfo["type"] == "winrm" {
		cmdargs = append(cmdargs, "-p", "\""+connInfo["password"]+"\"", "--no-ssl")
	} else {
		if sudo {
			cmdargs = append(cmdargs, "--run-as", "root")
		}

		cmdargs = append(cmdargs, "--no-host-key-check")
	}

	cmdargs = append(cmdargs, "--format", "json", "--connect-timeout", "120", task)

	if args != nil {
		for key, value := range args {
			cmdargs = append(cmdargs, strings.Join([]string{key, value}, "="))
		}
	}

	out, err := runCommand(strings.Join(cmdargs, " "), timeout)
	if err != nil {
		return nil, fmt.Errorf("Bolt: \"%s\": %s: %s", strings.Join(cmdargs, " "), out, err)
	}

	result := new(Result)
	if err = json.Unmarshal(out, result); err != nil {
		return nil, err
	}

	return result, nil
}
