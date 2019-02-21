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

func (f fish) Install(cmd, bin string) error {
	completionFile := filepath.Join(f.configDir, "completions", fmt.Sprintf("%s.fish", cmd))
	completeCmd := f.cmd(cmd, bin)
	if _, err := os.Stat(completionFile); err == nil {
		return fmt.Errorf("already installed at %s", completionFile)
	}

	return createFile(completionFile, completeCmd)
}

func (f fish) Uninstall(cmd, bin string) error {
	completionFile := filepath.Join(f.configDir, "completions", fmt.Sprintf("%s.fish", cmd))
	if _, err := os.Stat(completionFile); err != nil {
		return fmt.Errorf("does not installed in %s", f.configDir)
	}

	return os.Remove(completionFile)
}

func (f fish) cmd(cmd, bin string) string {
	var buf bytes.Buffer
	params := struct{ Cmd, Bin string }{cmd, bin}
	template.Must(template.New("cmd").Parse(`
function __complete_{{.Cmd}}
    set -lx COMP_LINE (string join ' ' (commandline -o))
    test (commandline -ct) = ""
    and set COMP_LINE "$COMP_LINE "
    {{.Bin}}
end
complete -c {{.Cmd}} -a "(__complete_{{.Cmd}})"
`)).Execute(&buf, params)

	return string(buf.Bytes())
}
