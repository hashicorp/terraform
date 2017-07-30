package command

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/fmtcmd"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/hcl/token"
	"github.com/mitchellh/cli"
)

const (
	stdinArg      = "-"
	fileExtension = "tf"
	infinity      = 1 << 30
)

// populateFilter is a filter that auto-populates the descriptions of variables
// and outputs with leading comment data.
//
// The intended behaviour of this filter is to use it to take data from the
// first contiguous block of text from a leading comment for a variable or
// output, and either pre-populate, or update the description of an already
// existing description field with the new data.
type populateFilter struct{}

// Filter implements HCL's printer.Filter for populateFilter.
func (f *populateFilter) Filter(n *ast.File) error {
	for _, obj := range n.Node.(*ast.ObjectList).Items {
		if obj.LeadComment != nil && obj.Keys[0].Token.Type == token.IDENT && (obj.Keys[0].Token.Text == "variable" || obj.Keys[0].Token.Text == "output") {
			// Insert the first line of the lead comment into the description. The
			// "first line" in this context means the first contiguous block before a
			// blank in the comments (the "basic" description).
			var lines []string
			for _, l := range obj.LeadComment.List {
				// Split on line breaks and iterate on that. This may seem frivolous,
				// but in slash-star comment forms, the entire comment block is treated
				// as a single comment, hence this needs to be split here to avoid
				// attempting to insert the entire comment, along with any prefixing
				// comment markers.
				for _, s := range strings.Split(l.Text, "\n") {
					r := regexp.MustCompile("^\\s*(?:[/*]+|#+)")
					s = r.ReplaceAllLiteralString(s, "")
					s = strings.TrimSpace(s)
					if s == "" {
						// Comment line composed of entirely whitespace, end of description,
						// unless this there has been no text data added yet (line could have
						// been a heading rule/divider).
						if len(lines) < 1 {
							continue
						}
						// We are done parsing the first block - we need to break out of
						// both loops.
						goto done
					}
					lines = append(lines, s)
				}
			}
		done:
			newVal := fmt.Sprintf("\"%s\"", strings.Join(lines, " "))
			if newVal == "" {
				// No string, abort (possibly just an empty lead comment)
				return nil
			}

			// Determine if there is already a description object first. If there is,
			// we just update the data.
			for _, item := range obj.Val.(*ast.ObjectType).List.Items {
				if item.Keys[0].Token.Type == token.IDENT && item.Keys[0].Token.Text == "description" {
					// Update description with new text.
					item.Val.(*ast.LiteralType).Token.Text = newVal
					// Update the LineComment, if it exists, with the new position - the
					// position is the start + len(newVal)+1.
					if item.LineComment != nil {
						item.LineComment.List[0].Start.Column = item.Val.(*ast.LiteralType).Token.Pos.Column + len(newVal) + 1
					}
					// Done
					return nil
				}
			}

			// If there is no description, we add one. Not just that, we add it at
			// the beginning of the object, with no position (save the Assignment
			// item, which needs a position to be rendered - this is given infinity).
			// This renders it as the first item with a blank line after it, which
			// ultimately looks nicer.
			desc := &ast.ObjectItem{
				Keys: []*ast.ObjectKey{
					&ast.ObjectKey{
						Token: token.Token{
							Type: token.IDENT,
							Text: "description",
						},
					},
				},
				Assign: token.Pos{Line: infinity},
				Val: &ast.LiteralType{
					Token: token.Token{
						Type: token.STRING,
						Text: newVal,
					},
				},
			}
			obj.Val.(*ast.ObjectType).List.Items = append([]*ast.ObjectItem{desc}, obj.Val.(*ast.ObjectType).List.Items...)
		}
	}
	return nil
}

// FmtCommand is a Command implementation that rewrites Terraform config
// files to a canonical format and style.
type FmtCommand struct {
	Meta
	opts  fmtcmd.Options
	input io.Reader // STDIN if nil
}

func (c *FmtCommand) Run(args []string) int {
	if c.input == nil {
		c.input = os.Stdin
	}

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	var populate bool

	cmdFlags := flag.NewFlagSet("fmt", flag.ContinueOnError)
	cmdFlags.BoolVar(&c.opts.List, "list", true, "list")
	cmdFlags.BoolVar(&c.opts.Write, "write", true, "write")
	cmdFlags.BoolVar(&c.opts.Diff, "diff", false, "diff")
	cmdFlags.BoolVar(&populate, "populate", false, "populate")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The fmt command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	}

	var dirs []string
	if len(args) == 0 {
		dirs = []string{"."}
	} else if args[0] == stdinArg {
		c.opts.List = false
		c.opts.Write = false
	} else {
		dirs = []string{args[0]}
	}

	output := &cli.UiWriter{Ui: c.Ui}
	if populate {
		c.opts.Filters = []printer.Filter{&populateFilter{}}
	}
	err = fmtcmd.Run(dirs, []string{fileExtension}, c.input, output, c.opts)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running fmt: %s", err))
		return 2
	}

	return 0
}

func (c *FmtCommand) Help() string {
	helpText := `
Usage: terraform fmt [options] [DIR]

	Rewrites all Terraform configuration files to a canonical format.

	If DIR is not specified then the current working directory will be used.
	If DIR is "-" then content will be read from STDIN.

Options:

  -list=true       List files whose formatting differs (always false if using STDIN)

  -write=true      Write result to source file instead of STDOUT (always false if using STDIN)

  -diff=false      Display diffs of formatting changes
  
	-populate=false  Auto-populate variable and output descriptions with first
	                 contiguous block of leading comment data

`
	return strings.TrimSpace(helpText)
}

func (c *FmtCommand) Synopsis() string {
	return "Rewrites config files to canonical format"
}
