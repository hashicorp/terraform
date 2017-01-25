package gottyclient

import (
	"errors"
	"os"
)

func notifySignalSIGWINCH(c chan<- os.Signal) {
}

func resetSignalSIGWINCH() {
}

func syscallTIOCGWINSZ() ([]byte, error) {
	return nil, errors.New("SIGWINCH isn't supported on this ARCH")
}
