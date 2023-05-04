// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows
// +build windows

package terminal

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/windows"

	// We're continuing to use this third-party library on Windows because it
	// has the additional IsCygwinTerminal function, which includes some useful
	// heuristics for recognizing when a pipe seems to be connected to a
	// legacy terminal emulator on Windows versions that lack true pty support.
	// We now use golang.org/x/term's functionality on other platforms.
	isatty "github.com/mattn/go-isatty"
)

func configureOutputHandle(f *os.File) (*OutputStream, error) {
	ret := &OutputStream{
		File: f,
	}

	if fd := f.Fd(); isatty.IsTerminal(fd) {
		// We have a few things to deal with here:
		// - Activating UTF-8 output support (mandatory)
		// - Activating virtual terminal support (optional)
		// These will not succeed on Windows 8 or early versions of Windows 10.

		// UTF-8 support means switching the console "code page" to CP_UTF8.
		// Notice that this doesn't take the specific file descriptor, because
		// the console is just ambiently associated with our process.
		err := SetConsoleOutputCP(CP_UTF8)
		if err != nil {
			return nil, fmt.Errorf("failed to set the console to UTF-8 mode; you may need to use a newer version of Windows: %s", err)
		}

		// If the console also allows us to turn on
		// ENABLE_VIRTUAL_TERMINAL_PROCESSING then we can potentially use VT
		// output, although the methods of Settings will make the final
		// determination on that because we might have some handles pointing at
		// terminals and other handles pointing at files/pipes.
		ret.getColumns = getColumnsWindowsConsole
		var mode uint32
		err = windows.GetConsoleMode(windows.Handle(fd), &mode)
		if err != nil {
			return ret, nil // We'll treat this as success but without VT support
		}
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
		err = windows.SetConsoleMode(windows.Handle(fd), mode)
		if err != nil {
			return ret, nil // We'll treat this as success but without VT support
		}

		// If we get here then we've successfully turned on VT processing, so
		// we can return an OutputStream that answers true when asked if it
		// is a Terminal.
		ret.isTerminal = staticTrue
		return ret, nil

	} else if isatty.IsCygwinTerminal(fd) {
		// Cygwin terminals -- and other VT100 "fakers" for older versions of
		// Windows -- are not really terminals in the usual sense, but rather
		// are pipes between the child process (Terraform) and the terminal
		// emulator. isatty.IsCygwinTerminal uses some heuristics to
		// distinguish those pipes from other pipes we might see if the user
		// were, for example, using the | operator on the command line.
		// If we get in here then we'll assume that we can send VT100 sequences
		// to this stream, even though it isn't a terminal in the usual sense.

		ret.isTerminal = staticTrue
		// TODO: Is it possible to detect the width of these fake terminals?
		return ret, nil
	}

	// If we fall out here then we have a non-terminal filehandle, so we'll
	// just accept all of the default OutputStream behaviors
	return ret, nil
}

func configureInputHandle(f *os.File) (*InputStream, error) {
	ret := &InputStream{
		File: f,
	}

	if fd := f.Fd(); isatty.IsTerminal(fd) {
		// We have to activate UTF-8 input, or else we fail. This will not
		// succeed on Windows 8 or early versions of Windows 10.
		// Notice that this doesn't take the specific file descriptor, because
		// the console is just ambiently associated with our process.
		err := SetConsoleCP(CP_UTF8)
		if err != nil {
			return nil, fmt.Errorf("failed to set the console to UTF-8 mode; you may need to use a newer version of Windows: %s", err)
		}
		ret.isTerminal = staticTrue
		return ret, nil
	} else if isatty.IsCygwinTerminal(fd) {
		// As with the output handles above, we'll use isatty's heuristic to
		// pretend that a pipe from mintty or a similar userspace terminal
		// emulator is actually a terminal.
		ret.isTerminal = staticTrue
		return ret, nil
	}

	// If we fall out here then we have a non-terminal filehandle, so we'll
	// just accept all of the default InputStream behaviors
	return ret, nil
}

func getColumnsWindowsConsole(f *os.File) int {
	// We'll just unconditionally ask the given file for its console buffer
	// info here, and let it fail if the file isn't actually a console.
	// (In practice, the init functions above only hook up this function
	// if the handle looks like a console, so this should succeed.)
	var info windows.ConsoleScreenBufferInfo
	err := windows.GetConsoleScreenBufferInfo(windows.Handle(f.Fd()), &info)
	if err != nil {
		return defaultColumns
	}
	return int(info.Size.X)
}

// Unfortunately not all of the Windows kernel functions we need are in
// x/sys/windows at the time of writing, so we need to call some of them
// directly. (If you're maintaining this in future and have the capacity to
// test it well, consider checking if these functions have been added upstream
// yet and switch to their wrapper stubs if so.
var modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
var procSetConsoleCP = modkernel32.NewProc("SetConsoleCP")
var procSetConsoleOutputCP = modkernel32.NewProc("SetConsoleOutputCP")

const CP_UTF8 = 65001

// (These are written in the style of the stubs in x/sys/windows, which is
// a little non-idiomatic just due to the awkwardness of the low-level syscall
// interface.)

func SetConsoleCP(codepageID uint32) (err error) {
	r1, _, e1 := syscall.Syscall(procSetConsoleCP.Addr(), 1, uintptr(codepageID), 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

func SetConsoleOutputCP(codepageID uint32) (err error) {
	r1, _, e1 := syscall.Syscall(procSetConsoleOutputCP.Addr(), 1, uintptr(codepageID), 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

func staticTrue(f *os.File) bool {
	return true
}

func staticFalse(f *os.File) bool {
	return false
}
