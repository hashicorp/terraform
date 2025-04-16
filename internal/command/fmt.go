// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

	"github.com/hashicorp/cli"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	stdinArg = "-"
)

var (
	fmtSupportedExts = []string{
		".tf",
		".tfvars",
		".tftest.hcl",
		".tfmock.hcl",
	}
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
	dos2unix  bool
	input     io.Reader // STDIN if nil
}

func (c *FmtCommand) Run(args []string) int {
	if c.input == nil {
		c.input = os.Stdin
	}

	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("fmt")
	cmdFlags.BoolVar(&c.list, "list", true, "list")
	cmdFlags.BoolVar(&c.write, "write", true, "write")
	cmdFlags.BoolVar(&c.diff, "diff", false, "diff")
	cmdFlags.BoolVar(&c.check, "check", false, "check")
	cmdFlags.BoolVar(&c.recursive, "recursive", false, "recursive")
	cmdFlags.BoolVar(&c.dos2unix, "dos2unix", false, "dos2unix")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()

	var paths []string
	if len(args) == 0 {
		paths = []string{"."}
	} else if args[0] == stdinArg {
		c.list = false
		c.write = false
	} else {
		paths = args
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
			fmtd := false
			for _, ext := range fmtSupportedExts {
				if strings.HasSuffix(path, ext) {
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

					// Take note that we processed the file.
					fmtd = true

					// Don't check the remaining extensions.
					break
				}
			}

			if !fmtd {
				diags = diags.Append(fmt.Errorf("Only .tf, .tfvars, and .tftest.hcl files can be processed with terraform fmt"))
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

	// Register this path as a synthetic configuration source, so that any
	// diagnostic errors can include the source code snippet
	c.registerSynthConfigSource(path, src)

	// File must be parseable as HCL native syntax before we'll try to format
	// it. If not, the formatter is likely to make drastic changes that would
	// be hard for the user to undo.
	_, syntaxDiags := hclsyntax.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if syntaxDiags.HasErrors() {
		diags = diags.Append(syntaxDiags)
		return diags
	}

	result := c.formatSourceCode(src, path)

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

		for _, ext := range fmtSupportedExts {
			if strings.HasSuffix(name, ext) {
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

				// Don't need to check the remaining extensions.
				break
			}
		}
	}

	return diags
}

// formatSourceCode is the formatting logic itself, applied to each file that
// is selected (directly or indirectly) on the command line.
func (c *FmtCommand) formatSourceCode(src []byte, filename string) []byte {
	f, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		// It would be weird to get here because the caller should already have
		// checked for syntax errors and returned them. We'll just do nothing
		// in this case, returning the input exactly as given.
		return src
	}

	c.formatBody(f.Body(), nil)
	c.dos2unix(f.Body(), nil)

	return f.Bytes()
}

func (c *FmtCommand) dos2unix(body *hclwrite.Body, inBlocks []string) {
	// Replace all occurrences of CRLF with LF
	return outputBytes := bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
}
func (c *FmtCommand) formatBody(body *hclwrite.Body, inBlocks []string) {
	attrs := body.Attributes()
	for name, attr := range attrs {
		if len(inBlocks) == 1 && inBlocks[0] == "variable" && name == "type" {
			cleanedExprTokens := c.formatTypeExpr(attr.Expr().BuildTokens(nil))
			body.SetAttributeRaw(name, cleanedExprTokens)
			continue
		}
		cleanedExprTokens := c.formatValueExpr(attr.Expr().BuildTokens(nil))
		body.SetAttributeRaw(name, cleanedExprTokens)
	}

	blocks := body.Blocks()
	for _, block := range blocks {
		// Normalize the label formatting, removing any weird stuff like
		// interleaved inline comments and using the idiomatic quoted
		// label syntax.
		block.SetLabels(block.Labels())

		inBlocks := append(inBlocks, block.Type())
		c.formatBody(block.Body(), inBlocks)
	}
}

func (c *FmtCommand) formatValueExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) < 5 {
		// Can't possibly be a "${ ... }" sequence without at least enough
		// tokens for the delimiters and one token inside them.
		return tokens
	}
	oQuote := tokens[0]
	oBrace := tokens[1]
	cBrace := tokens[len(tokens)-2]
	cQuote := tokens[len(tokens)-1]
	if oQuote.Type != hclsyntax.TokenOQuote || oBrace.Type != hclsyntax.TokenTemplateInterp || cBrace.Type != hclsyntax.TokenTemplateSeqEnd || cQuote.Type != hclsyntax.TokenCQuote {
		// Not an interpolation sequence at all, then.
		return tokens
	}

	inside := tokens[2 : len(tokens)-2]

	// We're only interested in sequences that are provable to be single
	// interpolation sequences, which we'll determine by hunting inside
	// the interior tokens for any other interpolation sequences. This is
	// likely to produce false negatives sometimes, but that's better than
	// false positives and we're mainly interested in catching the easy cases
	// here.
	quotes := 0
	for _, token := range inside {
		if token.Type == hclsyntax.TokenOQuote {
			quotes++
			continue
		}
		if token.Type == hclsyntax.TokenCQuote {
			quotes--
			continue
		}
		if quotes > 0 {
			// Interpolation sequences inside nested quotes are okay, because
			// they are part of a nested expression.
			// "${foo("${bar}")}"
			continue
		}
		if token.Type == hclsyntax.TokenTemplateInterp || token.Type == hclsyntax.TokenTemplateSeqEnd {
			// We've found another template delimiter within our interior
			// tokens, which suggests that we've found something like this:
			// "${foo}${bar}"
			// That isn't unwrappable, so we'll leave the whole expression alone.
			return tokens
		}
		if token.Type == hclsyntax.TokenQuotedLit {
			// If there's any literal characters in the outermost
			// quoted sequence then it is not unwrappable.
			return tokens
		}
	}

	// If we got down here without an early return then this looks like
	// an unwrappable sequence, but we'll trim any leading and trailing
	// newlines that might result in an invalid result if we were to
	// naively trim something like this:
	// "${
	//    foo
	// }"
	trimmed := c.trimNewlines(inside)

	// Finally, we check if the unwrapped expression is on multiple lines. If
	// so, we ensure that it is surrounded by parenthesis to make sure that it
	// parses correctly after unwrapping. This may be redundant in some cases,
	// but is required for at least multi-line ternary expressions.
	isMultiLine := false
	hasLeadingParen := false
	hasTrailingParen := false
	for i, token := range trimmed {
		switch {
		case i == 0 && token.Type == hclsyntax.TokenOParen:
			hasLeadingParen = true
		case token.Type == hclsyntax.TokenNewline:
			isMultiLine = true
		case i == len(trimmed)-1 && token.Type == hclsyntax.TokenCParen:
			hasTrailingParen = true
		}
	}
	if isMultiLine && !(hasLeadingParen && hasTrailingParen) {
		wrapped := make(hclwrite.Tokens, 0, len(trimmed)+2)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenOParen,
			Bytes: []byte("("),
		})
		wrapped = append(wrapped, trimmed...)
		wrapped = append(wrapped, &hclwrite.Token{
			Type:  hclsyntax.TokenCParen,
			Bytes: []byte(")"),
		})

		return wrapped
	}

	return trimmed
}

func (c *FmtCommand) formatTypeExpr(tokens hclwrite.Tokens) hclwrite.Tokens {
	switch len(tokens) {
	case 1:
		kwTok := tokens[0]
		if kwTok.Type != hclsyntax.TokenIdent {
			// Not a single type keyword, then.
			return tokens
		}

		// Collection types without an explicit element type mean
		// the element type is "any", so we'll normalize that.
		switch string(kwTok.Bytes) {
		case "list", "map", "set":
			return hclwrite.Tokens{
				kwTok,
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("any"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			return tokens
		}

	case 3:
		// A pre-0.12 legacy quoted string type, like "string".
		oQuote := tokens[0]
		strTok := tokens[1]
		cQuote := tokens[2]
		if oQuote.Type != hclsyntax.TokenOQuote || strTok.Type != hclsyntax.TokenQuotedLit || cQuote.Type != hclsyntax.TokenCQuote {
			// Not a quoted string sequence, then.
			return tokens
		}

		// Because this quoted syntax is from Terraform 0.11 and
		// earlier, which didn't have the idea of "any" as an,
		// element type, we use string as the default element
		// type. That will avoid oddities if somehow the configuration
		// was relying on numeric values being auto-converted to
		// string, as 0.11 would do. This mimicks what terraform
		// 0.12upgrade used to do, because we'd found real-world
		// modules that were depending on the auto-stringing.)
		switch string(strTok.Bytes) {
		case "string":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
			}
		case "list":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("list"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		case "map":
			return hclwrite.Tokens{
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("map"),
				},
				{
					Type:  hclsyntax.TokenOParen,
					Bytes: []byte("("),
				},
				{
					Type:  hclsyntax.TokenIdent,
					Bytes: []byte("string"),
				},
				{
					Type:  hclsyntax.TokenCParen,
					Bytes: []byte(")"),
				},
			}
		default:
			// Something else we're not expecting, then.
			return tokens
		}
	default:
		return tokens
	}
}

func (c *FmtCommand) trimNewlines(tokens hclwrite.Tokens) hclwrite.Tokens {
	if len(tokens) == 0 {
		return nil
	}
	var start, end int
	for start = 0; start < len(tokens); start++ {
		if tokens[start].Type != hclsyntax.TokenNewline {
			break
		}
	}
	for end = len(tokens); end > 0; end-- {
		if tokens[end-1].Type != hclsyntax.TokenNewline {
			break
		}
	}
	return tokens[start:end]
}

func (c *FmtCommand) Help() string {
	helpText := `
Usage: terraform [global options] fmt [options] [target...]

  Rewrites all Terraform configuration files to a canonical format. All
  configuration files (.tf), variables files (.tfvars), and testing files 
  (.tftest.hcl) are updated. JSON files (.tf.json, .tfvars.json, or 
  .tftest.json) are not modified.

  By default, fmt scans the current directory for configuration files. If you
  provide a directory for the target argument, then fmt will scan that
  directory instead. If you provide a file, then fmt will process just that
  file. If you provide a single dash ("-"), then fmt will read from standard
  input (STDIN).

  The content must be in the Terraform language native syntax; JSON is not
  supported.

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
	return "Reformat your configuration in the standard style"
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
