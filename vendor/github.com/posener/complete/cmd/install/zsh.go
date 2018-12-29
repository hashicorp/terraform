package install

import "fmt"

// (un)install in zsh
// basically adds/remove from .zshrc:
//
// autoload -U +X bashcompinit && bashcompinit"
// complete -C </path/to/completion/command> <command>
type zsh struct {
	rc string
}

func (z zsh) Install(cmd, bin string) error {
	completeCmd := z.cmd(cmd, bin)
	if lineInFile(z.rc, completeCmd) {
		return fmt.Errorf("already installed in %s", z.rc)
	}

	bashCompInit := "autoload -U +X bashcompinit && bashcompinit"
	if !lineInFile(z.rc, bashCompInit) {
		completeCmd = bashCompInit + "\n" + completeCmd
	}

	return appendToFile(z.rc, completeCmd)
}

func (z zsh) Uninstall(cmd, bin string) error {
	completeCmd := z.cmd(cmd, bin)
	if !lineInFile(z.rc, completeCmd) {
		return fmt.Errorf("does not installed in %s", z.rc)
	}

	return removeFromFile(z.rc, completeCmd)
}

func (zsh) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -o nospace -C %s %s", bin, cmd)
}
