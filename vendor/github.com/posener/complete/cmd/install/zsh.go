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

func (z zsh) IsInstalled(cmd, bin string) bool {
	completeCmd := z.cmd(cmd, bin)
	return lineInFile(z.rc, completeCmd)
}

func (z zsh) Install(cmd, bin string) error {
	if z.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed in %s", z.rc)
	}

	completeCmd := z.cmd(cmd, bin)
	bashCompInit := "autoload -U +X bashcompinit && bashcompinit"
	if !lineInFile(z.rc, bashCompInit) {
		completeCmd = bashCompInit + "\n" + completeCmd
	}

	return appendToFile(z.rc, completeCmd)
}

func (z zsh) Uninstall(cmd, bin string) error {
	if !z.IsInstalled(cmd, bin) {
		return fmt.Errorf("does not installed in %s", z.rc)
	}

	completeCmd := z.cmd(cmd, bin)
	return removeFromFile(z.rc, completeCmd)
}

func (zsh) cmd(cmd, bin string) string {
	return fmt.Sprintf("complete -o nospace -C %s %s", bin, cmd)
}
