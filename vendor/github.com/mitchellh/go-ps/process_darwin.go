// +build darwin

package ps

// #include "process_darwin.h"
import "C"

import (
	"sync"
)

// This lock is what verifies that C calling back into Go is only
// modifying data once at a time.
var darwinLock sync.Mutex
var darwinProcs []Process

type DarwinProcess struct {
	pid    int
	ppid   int
	binary string
}

func (p *DarwinProcess) Pid() int {
	return p.pid
}

func (p *DarwinProcess) PPid() int {
	return p.ppid
}

func (p *DarwinProcess) Executable() string {
	return p.binary
}

//export go_darwin_append_proc
func go_darwin_append_proc(pid C.pid_t, ppid C.pid_t, comm *C.char) {
	proc := &DarwinProcess{
		pid:    int(pid),
		ppid:   int(ppid),
		binary: C.GoString(comm),
	}

	darwinProcs = append(darwinProcs, proc)
}

func findProcess(pid int) (Process, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

func processes() ([]Process, error) {
	darwinLock.Lock()
	defer darwinLock.Unlock()
	darwinProcs = make([]Process, 0, 50)

	_, err := C.darwinProcesses()
	if err != nil {
		return nil, err
	}

	return darwinProcs, nil
}
