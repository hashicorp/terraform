package json

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl/v2"
)

// Parse attempts to parse the given buffer as JSON and, if successful, returns
// a hcl.File for the HCL configuration represented by it.
//
// This is not a generic JSON parser. Instead, it deals only with the profile
// of JSON used to express HCL configuration.
//
// The returned file is valid only if the returned diagnostics returns false
// from its HasErrors method. If HasErrors returns true, the file represents
// the subset of data that was able to be parsed, which may be none.
func Parse(src []byte, filename string) (*hcl.File, hcl.Diagnostics) {
	rootNode, diags := parseFileContent(src, filename)

	switch rootNode.(type) {
	case *objectVal, *arrayVal:
		// okay
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Root value must be object",
			Detail:   "The root value in a JSON-based configuration must be either a JSON object or a JSON array of objects.",
			Subject:  rootNode.StartRange().Ptr(),
		})

		// Since we've already produced an error message for this being
		// invalid, we'll return an empty placeholder here so that trying to
		// extract content from our root body won't produce a redundant
		// error saying the same thing again in more general terms.
		fakePos := hcl.Pos{
			Byte:   0,
			Line:   1,
			Column: 1,
		}
		fakeRange := hcl.Range{
			Filename: filename,
			Start:    fakePos,
			End:      fakePos,
		}
		rootNode = &objectVal{
			Attrs:     []*objectAttr{},
			SrcRange:  fakeRange,
			OpenRange: fakeRange,
		}
	}

	file := &hcl.File{
		Body: &body{
			val: rootNode,
		},
		Bytes: src,
		Nav:   navigation{rootNode},
	}
	return file, diags
}

// ParseFile is a convenience wrapper around Parse that first attempts to load
// data from the given filename, passing the result to Parse if successful.
//
// If the file cannot be read, an error diagnostic with nil context is returned.
func ParseFile(filename string) (*hcl.File, hcl.Diagnostics) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to open file",
				Detail:   fmt.Sprintf("The file %q could not be opened.", filename),
			},
		}
	}
	defer f.Close()

	src, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read file",
				Detail:   fmt.Sprintf("The file %q was opened, but an error occured while reading it.", filename),
			},
		}
	}

	return Parse(src, filename)
}
