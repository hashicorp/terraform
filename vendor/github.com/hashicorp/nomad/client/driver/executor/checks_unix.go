// +build darwin dragonfly freebsd linux netbsd openbsd solaris

package executor

import (
	"os/exec"
	"syscall"
)

func (e *ExecScriptCheck) setChroot(cmd *exec.Cmd) {
	if e.FSIsolation {
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.Chroot = e.taskDir
	}
	cmd.Dir = "/"
}
