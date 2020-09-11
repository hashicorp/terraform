package install

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-multierror"
)

type installer interface {
	IsInstalled(cmd, bin string) bool
	Install(cmd, bin string) error
	Uninstall(cmd, bin string) error
}

// Install complete command given:
// cmd: is the command name
func Install(cmd string) error {
	is := installers()
	if len(is) == 0 {
		return errors.New("Did not find any shells to install")
	}
	bin, err := getBinaryPath()
	if err != nil {
		return err
	}

	for _, i := range is {
		errI := i.Install(cmd, bin)
		if errI != nil {
			err = multierror.Append(err, errI)
		}
	}

	return err
}

// IsInstalled returns true if the completion
// for the given cmd is installed.
func IsInstalled(cmd string) bool {
	bin, err := getBinaryPath()
	if err != nil {
		return false
	}

	for _, i := range installers() {
		installed := i.IsInstalled(cmd, bin)
		if installed {
			return true
		}
	}

	return false
}

// Uninstall complete command given:
// cmd: is the command name
func Uninstall(cmd string) error {
	is := installers()
	if len(is) == 0 {
		return errors.New("Did not find any shells to uninstall")
	}
	bin, err := getBinaryPath()
	if err != nil {
		return err
	}

	for _, i := range is {
		errI := i.Uninstall(cmd, bin)
		if errI != nil {
			err = multierror.Append(err, errI)
		}
	}

	return err
}

func installers() (i []installer) {
	// The list of bash config files candidates where it is
	// possible to install the completion command.
	var bashConfFiles []string
	switch runtime.GOOS {
	case "darwin":
		bashConfFiles = []string{".bash_profile"}
	default:
		bashConfFiles = []string{".bashrc", ".bash_profile", ".bash_login", ".profile"}
	}
	for _, rc := range bashConfFiles {
		if f := rcFile(rc); f != "" {
			i = append(i, bash{f})
			break
		}
	}
	if f := rcFile(".zshrc"); f != "" {
		i = append(i, zsh{f})
	}
	if d := fishConfigDir(); d != "" {
		i = append(i, fish{d})
	}
	return
}

func fishConfigDir() string {
	configDir := filepath.Join(getConfigHomePath(), "fish")
	if configDir == "" {
		return ""
	}
	if info, err := os.Stat(configDir); err != nil || !info.IsDir() {
		return ""
	}
	return configDir
}

func getConfigHomePath() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		return filepath.Join(u.HomeDir, ".config")
	}
	return configHome
}

func getBinaryPath() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(bin)
}

func rcFile(name string) string {
	u, err := user.Current()
	if err != nil {
		return ""
	}
	path := filepath.Join(u.HomeDir, name)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
