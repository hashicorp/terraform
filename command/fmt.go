package command

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

const (
	stdinArg = "-"
)

// FmtCommand is a Command implementation that rewrites Terraform config
// files to a canonical format and style.
type FmtCommand struct {
	Meta
	list      bool
	write     bool
	diff      bool
	check     bool
	recursive bool
	input     io.Reader // STDIN if nil
}

func (c *FmtCommand) Run(args []string) int {
	if c.input == nil {
		c.input = os.Stdin
	}

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("fmt")
	cmdFlags.BoolVar(&c.list, "list", true, "list")
	cmdFlags.BoolVar(&c.write, "write", true, "write")
	cmdFlags.BoolVar(&c.diff, "diff", false, "diff")
	cmdFlags.BoolVar(&c.check, "check", false, "check")
	cmdFlags.BoolVar(&c.recursive, "recursive", false, "recursive")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The fmt command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	}

	var paths []string
	if len(args) == 0 {
		paths = []string{"."}
	} else if args[0] == stdinArg {
		c.list = false
		c.write = false
	} else {
		paths = []string{args[0]}
	}

	var output io.Writer
	list := c.list // preserve the original value of -list
	if c.check {
		// set to true so we can use the list output to check
		// if the input needs formatting
		c.list = true
		c.write = false
		output = &bytes.Buffer{}
	} else {
		output = &cli.UiWriter{Ui: c.Ui}
	}

	diags := c.fmt(paths, c.input, output)
	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 2
	}

	if c.check {
		buf := output.(*bytes.Buffer)
		ok := buf.Len() == 0
		if list {
			io.Copy(&cli.UiWriter{Ui: c.Ui}, buf)
		}
		if ok {
			return 0
		} else {
			return 3
		}
	}

	return 0
}

func (c *FmtCommand) fmt(paths []string, stdin io.Reader, stdout io.Writer) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if len(paths) == 0 { // Assuming stdin, then.
		if c.write {
			diags = diags.Append(fmt.Errorf("Option -write cannot be used when reading from stdin"))
			return diags
		}
		fileDiags := c.processFile("<stdin>", stdin, stdout, true)
		diags = diags.Append(fileDiags)
		return diags
	}

	for _, path := range paths {
		path = c.normalizePath(path)
		info, err := os.Stat(path)
		if err != nil {
			diags = diags.Append(fmt.Errorf("No file or directory at %s", path))
			return diags
		}
		if info.IsDir() {
			dirDiags := c.processDir(path, stdout)
			diags = diags.Append(dirDiags)
		} else {
			switch filepath.Ext(path) {
			case ".tf", ".tfvars":
				f, err := os.Open(path)
				if err != nil {
					// Open does not produce error messages that are end-user-appropriate,
					// so we'll need to simplify here.
					diags = diags.Append(fmt.Errorf("Failed to read file %s", path))
					continue
				}

				fileDiags := c.processFile(c.normalizePath(path), f, stdout, false)
				diags = diags.Append(fileDiags)
				f.Close()
			default:
				diags = diags.Append(fmt.Errorf("Only .tf and .tfvars files can be processed with terraform fmt"))
				continue
			}
		}
	}

	return diags
}

func (c *FmtCommand) processFile(path string, r io.Reader, w io.Writer, isStdout bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	log.Printf("[TRACE] terraform fmt: Formatting %s", path)

	src, err := ioutil.ReadAll(r)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to read %s", path))
		return diags
	}

	// File must be parseable as HCL native syntax before we'll try to format
	// it. If not, the formatter is likely to make drastic changes that would
	// be hard for the user to undo.
	_, syntaxDiags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if syntaxDiags.HasErrors() {
		diags = diags.Append(syntaxDiags)
		return diags
	}

	result := hclwrite.Format(src)

	if !bytes.Equal(src, result) {
		// Something was changed
		if c.list {
			fmt.Fprintln(w, path)
		}
		if c.write {
			err := ioutil.WriteFile(path, result, 0644)
			if err != nil {
				diags = diags.Append(fmt.Errorf("Failed to write %s", path))
				return diags
			}
		}
		if c.diff {
			diff, err := bytesDiff(src, result, path)
			if err != nil {
				diags = diags.Append(fmt.Errorf("Failed to generate diff for %s: %s", path, err))
				return diags
			}
			w.Write(diff)
		}
	}

	if !c.list && !c.write && !c.diff {
		_, err = w.Write(result)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to write result"))
		}
	}

	return diags
}

func (c *FmtCommand) processDir(path string, stdout io.Writer) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	log.Printf("[TRACE] terraform fmt: looking for files in %s", path)

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		switch {
		case os.IsNotExist(err):
			diags = diags.Append(fmt.Errorf("There is no configuration directory at %s", path))
		default:
			// ReadDir does not produce error messages that are end-user-appropriate,
			// so we'll need to simplify here.
			diags = diags.Append(fmt.Errorf("Cannot read directory %s", path))
		}
		return diags
	}

	for _, info := range entries {
		name := info.Name()
		if configs.IsIgnoredFile(name) {
			continue
		}
		subPath := filepath.Join(path, name)
		if info.IsDir() {
			if c.recursive {
				subDiags := c.processDir(subPath, stdout)
				diags = diags.Append(subDiags)
			}

			// We do not recurse into child directories by default because we
			// want to mimic the file-reading behavior of "terraform plan", etc,
			// operating on one module at a time.
			continue
		}

		ext := filepath.Ext(name)
		switch ext {
		case ".tf", ".tfvars":
			f, err := os.Open(subPath)
			if err != nil {
				// Open does not produce error messages that are end-user-appropriate,
				// so we'll need to simplify here.
				diags = diags.Append(fmt.Errorf("Failed to read file %s", subPath))
				continue
			}

			fileDiags := c.processFile(c.normalizePath(subPath), f, stdout, false)
			diags = diags.Append(fileDiags)
			f.Close()
		}
	}

	return diags
}

func (c *FmtCommand) Help() string {
	helpText := `
Usage: terraform fmt [options] [DIR]

	Rewrites all Terraform configuration files to a canonical format. Both
	configuration files (.tf) and variables files (.tfvars) are updated.
	JSON files (.tf.json or .tfvars.json) are not modified.

	If DIR is not specified then the current working directory will be used.
	If DIR is "-" then content will be read from STDIN. The given content must
	be in the Terraform language native syntax; JSON is not supported.

Options:

  -list=false    Don't list files whose formatting differs
                 (always disabled if using STDIN)

  -write=false   Don't write to source files
                 (always disabled if using STDIN or -check)

  -diff          Display diffs of formatting changes

  -check         Check if the input is formatted. Exit status will be 0 if all
                 input is properly formatted and non-zero otherwise.

  -no-color      If specified, output won't contain any color.

  -recursive     Also process files in subdirectories. By default, only the
                 given directory (or current directory) is processed.
`
	return strings.TrimSpace(helpText)
}

func (c *FmtCommand) Synopsis() string {
	return "Rewrites config files to canonical format"
}

func bytesDiff(b1, b2 []byte, path string) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "--label=old/"+path, "--label=new/"+path, "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}
