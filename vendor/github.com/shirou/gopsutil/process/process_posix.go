// +build linux freebsd darwin

package process

import (
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/internal/common"
)

// POSIX
func getTerminalMap() (map[uint64]string, error) {
	ret := make(map[uint64]string)
	var termfiles []string

	d, err := os.Open("/dev")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	devnames, err := d.Readdirnames(-1)
	for _, devname := range devnames {
		if strings.HasPrefix(devname, "/dev/tty") {
			termfiles = append(termfiles, "/dev/tty/"+devname)
		}
	}

	ptsd, err := os.Open("/dev/pts")
	if err != nil {
		return nil, err
	}
	defer ptsd.Close()

	ptsnames, err := ptsd.Readdirnames(-1)
	for _, ptsname := range ptsnames {
		termfiles = append(termfiles, "/dev/pts/"+ptsname)
	}

	for _, name := range termfiles {
		stat := syscall.Stat_t{}
		if err = syscall.Stat(name, &stat); err != nil {
			return nil, err
		}
		rdev := uint64(stat.Rdev)
		ret[rdev] = strings.Replace(name, "/dev", "", -1)
	}
	return ret, nil
}

// SendSignal sends a syscall.Signal to the process.
// Currently, SIGSTOP, SIGCONT, SIGTERM and SIGKILL are supported.
func (p *Process) SendSignal(sig syscall.Signal) error {
	sigAsStr := "INT"
	switch sig {
	case syscall.SIGSTOP:
		sigAsStr = "STOP"
	case syscall.SIGCONT:
		sigAsStr = "CONT"
	case syscall.SIGTERM:
		sigAsStr = "TERM"
	case syscall.SIGKILL:
		sigAsStr = "KILL"
	}

	kill, err := exec.LookPath("kill")
	if err != nil {
		return err
	}
	cmd := exec.Command(kill, "-s", sigAsStr, strconv.Itoa(int(p.Pid)))
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	err = common.WaitTimeout(cmd, common.Timeout)
	if err != nil {
		return err
	}

	return nil
}

// Suspend sends SIGSTOP to the process.
func (p *Process) Suspend() error {
	return p.SendSignal(syscall.SIGSTOP)
}

// Resume sends SIGCONT to the process.
func (p *Process) Resume() error {
	return p.SendSignal(syscall.SIGCONT)
}

// Terminate sends SIGTERM to the process.
func (p *Process) Terminate() error {
	return p.SendSignal(syscall.SIGTERM)
}

// Kill sends SIGKILL to the process.
func (p *Process) Kill() error {
	return p.SendSignal(syscall.SIGKILL)
}

// Username returns a username of the process.
func (p *Process) Username() (string, error) {
	uids, err := p.Uids()
	if err != nil {
		return "", err
	}
	if len(uids) > 0 {
		u, err := user.LookupId(strconv.Itoa(int(uids[0])))
		if err != nil {
			return "", err
		}
		return u.Username, nil
	}
	return "", nil
}
