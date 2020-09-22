package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
)

type OutputTemplateData struct {
	Dir  string
	OS   string
	Arch string
}

type CompileOpts struct {
	PackagePath string
	Platform    Platform
	OutputTpl   string
	Ldflags     string
	Gcflags     string
	Asmflags    string
	Tags        string
	ModMode     string
	Cgo         bool
	Rebuild     bool
	GoCmd       string
}

// GoCrossCompile
func GoCrossCompile(opts *CompileOpts) error {
	env := append(os.Environ(),
		"GOOS="+opts.Platform.OS,
		"GOARCH="+opts.Platform.Arch)

	// If we're building for our own platform, then enable cgo always. We
	// respect the CGO_ENABLED flag if that is explicitly set on the platform.
	if !opts.Cgo && os.Getenv("CGO_ENABLED") != "0" {
		opts.Cgo = runtime.GOOS == opts.Platform.OS &&
			runtime.GOARCH == opts.Platform.Arch
	}

	// If cgo is enabled then set that env var
	if opts.Cgo {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	var outputPath bytes.Buffer
	tpl, err := template.New("output").Parse(opts.OutputTpl)
	if err != nil {
		return err
	}
	tplData := OutputTemplateData{
		Dir:  filepath.Base(opts.PackagePath),
		OS:   opts.Platform.OS,
		Arch: opts.Platform.Arch,
	}
	if err := tpl.Execute(&outputPath, &tplData); err != nil {
		return err
	}

	if opts.Platform.OS == "windows" {
		outputPath.WriteString(".exe")
	}

	// Determine the full path to the output so that we can change our
	// working directory when executing go build.
	outputPathReal := outputPath.String()
	outputPathReal, err = filepath.Abs(outputPathReal)
	if err != nil {
		return err
	}

	// Go prefixes the import directory with '_' when it is outside
	// the GOPATH.For this, we just drop it since we move to that
	// directory to build.
	chdir := ""
	if opts.PackagePath[0] == '_' {
		if runtime.GOOS == "windows" {
			// We have to replace weird paths like this:
			//
			//   _/c_/Users
			//
			// With:
			//
			//   c:\Users
			//
			re := regexp.MustCompile("^/([a-zA-Z])_/")
			chdir = re.ReplaceAllString(opts.PackagePath[1:], "$1:\\")
			chdir = strings.Replace(chdir, "/", "\\", -1)
		} else {
			chdir = opts.PackagePath[1:]
		}

		opts.PackagePath = ""
	}

	args := []string{"build"}
	if opts.Rebuild {
		args = append(args, "-a")
	}
	if opts.ModMode != "" {
		args = append(args, "-mod", opts.ModMode)
	}
	args = append(args,
		"-gcflags", opts.Gcflags,
		"-ldflags", opts.Ldflags,
		"-asmflags", opts.Asmflags,
		"-tags", opts.Tags,
		"-o", outputPathReal,
		opts.PackagePath)

	_, err = execGo(opts.GoCmd, env, chdir, args...)
	return err
}

// GoMainDirs returns the file paths to the packages that are "main"
// packages, from the list of packages given. The list of packages can
// include relative paths, the special "..." Go keyword, etc.
func GoMainDirs(packages []string, GoCmd string) ([]string, error) {
	args := make([]string, 0, len(packages)+3)
	args = append(args, "list", "-f", "{{.Name}}|{{.ImportPath}}")
	args = append(args, packages...)

	output, err := execGo(GoCmd, nil, "", args...)
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(output))
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			log.Printf("Bad line reading packages: %s", line)
			continue
		}

		if parts[0] == "main" {
			results = append(results, parts[1])
		}
	}

	return results, nil
}

// GoRoot returns the GOROOT value for the compiled `go` binary.
func GoRoot() (string, error) {
	output, err := execGo("go", nil, "", "env", "GOROOT")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// GoVersion reads the version of `go` that is on the PATH. This is done
// instead of `runtime.Version()` because it is possible to run gox against
// another Go version.
func GoVersion() (string, error) {
	// NOTE: We use `go run` instead of `go version` because the output
	// of `go version` might change whereas the source is guaranteed to run
	// for some time thanks to Go's compatibility guarantee.

	td, err := ioutil.TempDir("", "gox")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(td)

	// Write the source code for the program that will generate the version
	sourcePath := filepath.Join(td, "version.go")
	if err := ioutil.WriteFile(sourcePath, []byte(versionSource), 0644); err != nil {
		return "", err
	}

	// Execute and read the version, which will be the only thing on stdout.
	return execGo("go", nil, "", "run", sourcePath)
}

// GoVersionParts parses the version numbers from the version itself
// into major and minor: 1.5, 1.4, etc.
func GoVersionParts() (result [2]int, err error) {
	version, err := GoVersion()
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(version, "go%d.%d", &result[0], &result[1])
	return
}

func execGo(GoCmd string, env []string, dir string, args ...string) (string, error) {
	var stderr, stdout bytes.Buffer
	cmd := exec.Command(GoCmd, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if env != nil {
		cmd.Env = env
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf("%s\nStderr: %s", err, stderr.String())
		return "", err
	}

	return stdout.String(), nil
}

const versionSource = `package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Print(runtime.Version())
}`
