package hclparse

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/hcl2/hcl/json"
)

// NOTE: This is the public interface for parsing. The actual parsers are
// in other packages alongside this one, with this package just wrapping them
// to provide a unified interface for the caller across all supported formats.

// Parser is the main interface for parsing configuration files. As well as
// parsing files, a parser also retains a registry of all of the files it
// has parsed so that multiple attempts to parse the same file will return
// the same object and so the collected files can be used when printing
// diagnostics.
//
// Any diagnostics for parsing a file are only returned once on the first
// call to parse that file. Callers are expected to collect up diagnostics
// and present them together, so returning diagnostics for the same file
// multiple times would create a confusing result.
type Parser struct {
	files map[string]*hcl.File
}

// NewParser creates a new parser, ready to parse configuration files.
func NewParser() *Parser {
	return &Parser{
		files: map[string]*hcl.File{},
	}
}

// ParseHCL parses the given buffer (which is assumed to have been loaded from
// the given filename) as a native-syntax configuration file and returns the
// hcl.File object representing it.
func (p *Parser) ParseHCL(src []byte, filename string) (*hcl.File, hcl.Diagnostics) {
	if existing := p.files[filename]; existing != nil {
		return existing, nil
	}

	file, diags := hclsyntax.ParseConfig(src, filename, hcl.Pos{Byte: 0, Line: 1, Column: 1})
	p.files[filename] = file
	return file, diags
}

// ParseHCLFile reads the given filename and parses it as a native-syntax HCL
// configuration file. An error diagnostic is returned if the given file
// cannot be read.
func (p *Parser) ParseHCLFile(filename string) (*hcl.File, hcl.Diagnostics) {
	if existing := p.files[filename]; existing != nil {
		return existing, nil
	}

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read file",
				Detail:   fmt.Sprintf("The configuration file %q could not be read.", filename),
			},
		}
	}

	return p.ParseHCL(src, filename)
}

// ParseJSON parses the given JSON buffer (which is assumed to have been loaded
// from the given filename) and returns the hcl.File object representing it.
func (p *Parser) ParseJSON(src []byte, filename string) (*hcl.File, hcl.Diagnostics) {
	if existing := p.files[filename]; existing != nil {
		return existing, nil
	}

	file, diags := json.Parse(src, filename)
	p.files[filename] = file
	return file, diags
}

// ParseJSONFile reads the given filename and parses it as JSON, similarly to
// ParseJSON. An error diagnostic is returned if the given file cannot be read.
func (p *Parser) ParseJSONFile(filename string) (*hcl.File, hcl.Diagnostics) {
	if existing := p.files[filename]; existing != nil {
		return existing, nil
	}

	file, diags := json.ParseFile(filename)
	p.files[filename] = file
	return file, diags
}

// AddFile allows a caller to record in a parser a file that was parsed some
// other way, thus allowing it to be included in the registry of sources.
func (p *Parser) AddFile(filename string, file *hcl.File) {
	p.files[filename] = file
}

// Sources returns a map from filenames to the raw source code that was
// read from them. This is intended to be used, for example, to print
// diagnostics with contextual information.
//
// The arrays underlying the returned slices should not be modified.
func (p *Parser) Sources() map[string][]byte {
	ret := make(map[string][]byte)
	for fn, f := range p.files {
		ret[fn] = f.Bytes
	}
	return ret
}

// Files returns a map from filenames to the File objects produced from them.
// This is intended to be used, for example, to print diagnostics with
// contextual information.
//
// The returned map and all of the objects it refers to directly or indirectly
// must not be modified.
func (p *Parser) Files() map[string]*hcl.File {
	return p.files
}
