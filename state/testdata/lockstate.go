package main

import (
	"io"
	"log"
	"os"

	"github.com/hashicorp/terraform/state"
)

// Attempt to open and lock a terraform state file.
// Lock failure exits with 0 and writes "lock failed" to stderr.
func main() {
	if len(os.Args) != 2 {
		log.Fatal(os.Args[0], "statefile")
	}

	s := &state.LocalState{
		Path: os.Args[1],
	}

	info := state.NewLockInfo()
	info.Operation = "test"
	info.Info = "state locker"

	_, err := s.Lock(info)
	if err != nil {
		io.WriteString(os.Stderr, "lock failed")

	}

	return
}
