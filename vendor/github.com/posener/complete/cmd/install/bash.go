package install

import "fmt"

// (un)install in bash
// basically adds/remove from .bashrc:
//
// complete -C </path/to/completion/command> <command>
type bash struct {
	rc string
}

func (b bash) IsInstalled(cmd, bin string) bool {
	completeCmd := b.cmd(cmd, bin)
	return lineInFile(b.rc, completeCmd)
}

func (b bash) Install(cmd, bin string) error {
	if b.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed in %s", b.rc)
	}
	completeCmd := b.cmd(cmd, bin)
	return appendToFile(b.rc, completeCmd)
}

func (b bash) Uninstall(cmd, bin string) error {
	if !b.IsInstalled(cmd, bin) {
		return fmt.Errorf("does not installed in %s", b.rc)
	}

	completeCmd := b.cmd(cmd, bin)
	return removeFromFile(b.rc, completeCmd)
}

func (bash) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -C %s %s", bin, cmd)
}
