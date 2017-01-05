// +build windows

package terminal

import (
	"os"
	"syscall"
)

// see SetConsoleMode documentation for bit flags
// http://msdn.microsoft.com/en-us/library/windows/desktop/ms686033(v=vs.85).aspx
const ENABLE_ECHO_INPUT = 0x0004

func (ui terminalUI) AskForPassword(prompt string) (passwd string) {
	hStdin := syscall.Handle(os.Stdin.Fd())
	var originalMode uint32

	err := syscall.GetConsoleMode(hStdin, &originalMode)
	if err != nil {
		return
	}

	var newMode uint32 = (originalMode &^ ENABLE_ECHO_INPUT)

	err = setConsoleMode(hStdin, newMode)
	defer setConsoleMode(hStdin, originalMode)
	defer ui.Say("")

	if err != nil {
		return
	}

	return ui.Ask(prompt)
}

func setConsoleMode(console syscall.Handle, mode uint32) (err error) {
	dll := syscall.MustLoadDLL("kernel32")
	proc := dll.MustFindProc("SetConsoleMode")
	r, _, err := proc.Call(uintptr(console), uintptr(mode))

	if r == 0 {
		return err
	}
	return nil
}
