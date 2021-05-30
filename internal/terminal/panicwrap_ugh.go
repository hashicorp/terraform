package terminal

import "os"

// This file has some annoying nonsense to, yet again, work around the
// panicwrap hack.
//
// Specifically, typically when we're running Terraform the stderr handle is
// not directly connected to the terminal but is instead a pipe into a parent
// process gathering up the output just in case a panic message appears.
// However, this package needs to know whether the _real_ stderr is connected
// to a terminal and what its width is.
//
// To work around that, we'll first initialize the terminal in the parent
// process, and then capture information about stderr into an environment
// variable so we can pass it down to the child process. The child process
// will then use the environment variable to pretend that the panicwrap pipe
// has the same characteristics as the terminal that it's indirectly writing
// to.
//
// This file has some helpers for implementing that awkward handshake, but the
// handshake itself is in package main, interspersed with all of the other
// panicwrap machinery.
//
// You might think that the code in helper/wrappedstreams could avoid this
// problem, but that package is broken on Windows: it always fails to recover
// the real stderr, and it also gets an incorrect result if the user was
// redirecting or piping stdout/stdin. So... we have this hack instead, which
// gets a correct result even on Windows and even with I/O redirection.

// StateForAfterPanicWrap is part of the workaround for panicwrap that
// captures some characteristics of stderr that the caller can pass to the
// panicwrap child process somehow and then use ReinitInsidePanicWrap.
func (s *Streams) StateForAfterPanicWrap() *PrePanicwrapState {
	return &PrePanicwrapState{
		StderrIsTerminal: s.Stderr.IsTerminal(),
		StderrWidth:      s.Stderr.Columns(),
	}
}

// ReinitInsidePanicwrap is part of the workaround for panicwrap that
// produces a Streams containing a potentially-lying Stderr that might
// claim to be a terminal even if it's actually a pipe connected to the
// parent process.
//
// That's an okay lie in practice because the parent process will copy any
// data it recieves via that pipe verbatim to the real stderr anyway. (The
// original call to Init in the parent process should've already done any
// necessary modesetting on the Stderr terminal, if any.)
//
// The state argument can be nil if we're not running in panicwrap mode,
// in which case this function behaves exactly the same as Init.
func ReinitInsidePanicwrap(state *PrePanicwrapState) (*Streams, error) {
	ret, err := Init()
	if err != nil {
		return ret, err
	}
	if state != nil {
		// A lying stderr, then.
		ret.Stderr = &OutputStream{
			File: ret.Stderr.File,
			isTerminal: func(f *os.File) bool {
				return state.StderrIsTerminal
			},
			getColumns: func(f *os.File) int {
				return state.StderrWidth
			},
		}
	}
	return ret, nil
}

// PrePanicwrapState is a horrible thing we use to work around panicwrap,
// related to both Streams.StateForAfterPanicWrap and ReinitInsidePanicwrap.
type PrePanicwrapState struct {
	StderrIsTerminal bool
	StderrWidth      int
}
