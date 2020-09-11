package install

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// (un)install in fish

type fish struct {
	configDir string
}

func (f fish) IsInstalled(cmd, bin string) bool {
	completionFile := f.getCompletionFilePath(cmd)
	if _, err := os.Stat(completionFile); err == nil {
		return true
	}
	return false
}

func (f fish) Install(cmd, bin string) error {
	if f.IsInstalled(cmd, bin) {
		return fmt.Errorf("already installed at %s", f.getCompletionFilePath(cmd))
	}

	completionFile := f.getCompletionFilePath(cmd)
	completeCmd, err := f.cmd(cmd, bin)
	if err != nil {
		return err
	}

	return createFile(completionFile, completeCmd)
}

func (f fish) Uninstall(cmd, bin string) error {
	if !f.IsInstalled(cmd, bin) {
		return fmt.Errorf("does not installed in %s", f.configDir)
	}

	completionFile := f.getCompletionFilePath(cmd)
	return os.Remove(completionFile)
}

func (f fish) getCompletionFilePath(cmd string) string {
	return filepath.Join(f.configDir, "completions", fmt.Sprintf("%s.fish", cmd))
}

func (f fish) cmd(cmd, bin string) (string, error) {
	var buf bytes.Buffer
	params := struct{ Cmd, Bin string }{cmd, bin}
	tmpl := template.Must(template.New("cmd").Parse(`
function __complete_{{.Cmd}}
    set -lx COMP_LINE (commandline -cp)
    test -z (commandline -ct)
    and set COMP_LINE "$COMP_LINE "
    {{.Bin}}
end
complete -f -c {{.Cmd}} -a "(__complete_{{.Cmd}})"
`))
	err := tmpl.Execute(&buf, params)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
