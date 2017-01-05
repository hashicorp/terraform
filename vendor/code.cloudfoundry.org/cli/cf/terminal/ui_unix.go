// Copied from https://code.google.com/p/gopass/

// +build !windows

package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	. "code.cloudfoundry.org/cli/cf/i18n"
)

const (
	sttyArg0  = "/bin/stty"
	execCWDir = ""
)

// Tells the terminal to turn echo off.
var sttyArgvEOff = []string{"stty", "-echo"}

// Tells the terminal to turn echo on.
var sttyArgvEOn = []string{"stty", "echo"}

var ws syscall.WaitStatus

func (ui terminalUI) AskForPassword(prompt string) (passwd string) {
	sig := make(chan os.Signal, 10)

	// Display the prompt.
	fmt.Printf("\n%s%s ", prompt, PromptColor(">"))

	// File descriptors for stdin, stdout, and stderr.
	fd := []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}

	// Setup notifications of termination signals to channel sig, create a process to
	// watch for these signals so we can turn back on echo if need be.
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGKILL, syscall.SIGQUIT,
		syscall.SIGTERM)
	defer signal.Stop(sig)

	go catchSignal(fd, sig)

	pid, err := echoOff(fd)
	defer echoOn(fd)
	if err != nil {
		return
	}

	passwd = readPassword(pid)

	// Carriage return after the user input.
	fmt.Println("")

	return
}

func readPassword(pid int) string {
	rd := bufio.NewReader(os.Stdin)
	_, _ = syscall.Wait4(pid, &ws, 0, nil)

	line, err := rd.ReadString('\n')
	if err == nil {
		return strings.TrimSpace(line)
	}
	return ""
}

func echoOff(fd []uintptr) (int, error) {
	pid, err := syscall.ForkExec(sttyArg0, sttyArgvEOff, &syscall.ProcAttr{Dir: execCWDir, Files: fd})

	if err != nil {
		return 0, fmt.Errorf(T("failed turning off console echo for password entry:\n{{.ErrorDescription}}", map[string]interface{}{"ErrorDescription": err}))
	}

	return pid, nil
}

// echoOn turns back on the terminal echo.
func echoOn(fd []uintptr) {
	// Turn on the terminal echo.
	pid, e := syscall.ForkExec(sttyArg0, sttyArgvEOn, &syscall.ProcAttr{Dir: execCWDir, Files: fd})

	if e == nil {
		_, _ = syscall.Wait4(pid, &ws, 0, nil)
	}
}

// catchSignal tries to catch SIGKILL, SIGQUIT and SIGINT so that we can turn terminal
// echo back on before the program ends.  Otherwise the user is left with echo off on
// their terminal.
func catchSignal(fd []uintptr, sig chan os.Signal) {
	select {
	case <-sig:
		echoOn(fd)
		os.Exit(2)
	}
}
