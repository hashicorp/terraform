package json

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/hcl2/hcl"
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
	if _, ok := rootNode.(*objectVal); !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Root value must be object",
			Detail:   "The root value in a JSON-based configuration must be a JSON object.",
			Subject:  rootNode.StartRange().Ptr(),
		})
		// Put in a placeholder objectVal just so the caller always gets
		// a valid file, even if it appears empty. This is useful for callers
		// that are doing static analysis of possibly-erroneous source code,
		// which will try to process the returned file even if we return
		// diagnostics of severity error. This way, they'll get a file that
		// has an empty body rather than a body that panics when probed.
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
			Attrs:     map[string]*objectAttr{},
			SrcRange:  fakeRange,
			OpenRange: fakeRange,
		}
	}
	file := &hcl.File{
		Body: &body{
			obj: rootNode.(*objectVal),
		},
		Bytes: src,
		Nav:   navigation{rootNode.(*objectVal)},
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
