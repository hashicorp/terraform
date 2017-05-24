package discover

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/kardianos/osext"
)

// Checks the current executable, then $GOPATH/bin, and finally the CWD, in that
// order. If it can't be found, an error is returned.
func NomadExecutable() (string, error) {
	nomadExe := "nomad"
	if runtime.GOOS == "windows" {
		nomadExe = "nomad.exe"
	}

	// Check the current executable.
	bin, err := osext.Executable()
	if err != nil {
		return "", fmt.Errorf("Failed to determine the nomad executable: %v", err)
	}

	if filepath.Base(bin) == nomadExe {
		return bin, nil
	}

	// Check the $PATH
	if bin, err := exec.LookPath(nomadExe); err == nil {
		return bin, nil
	}

	// Check the $GOPATH.
	bin = filepath.Join(os.Getenv("GOPATH"), "bin", nomadExe)
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}

	// Check the CWD.
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Could not find Nomad executable (%v): %v", nomadExe, err)
	}

	bin = filepath.Join(pwd, nomadExe)
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}

	// Check CWD/bin
	bin = filepath.Join(pwd, "bin", nomadExe)
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}

	return "", fmt.Errorf("Could not find Nomad executable (%v)", nomadExe)
}
