package testutil

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"
)

func ExecCompatible(t *testing.T) {
	if runtime.GOOS != "linux" || syscall.Geteuid() != 0 {
		t.Skip("Test only available running as root on linux")
	}
}

func JavaCompatible(t *testing.T) {
	if runtime.GOOS == "linux" && syscall.Geteuid() != 0 {
		t.Skip("Test only available when running as root on linux")
	}
}

func QemuCompatible(t *testing.T) {
	// Check if qemu exists
	bin := "qemu-system-x86_64"
	if runtime.GOOS == "windows" {
		bin = "qemu-img"
	}
	_, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Skip("Must have Qemu installed for Qemu specific tests to run")
	}
}

func RktCompatible(t *testing.T) {
	if runtime.GOOS == "windows" || syscall.Geteuid() != 0 {
		t.Skip("Must be root on non-windows environments to run test")
	}
	// else see if rkt exists
	_, err := exec.Command("rkt", "version").CombinedOutput()
	if err != nil {
		t.Skip("Must have rkt installed for rkt specific tests to run")
	}
}

func MountCompatible(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support mount")
	}

	if syscall.Geteuid() != 0 {
		t.Skip("Must be root to run test")
	}
}
