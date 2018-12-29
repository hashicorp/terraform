package hclsyntax

import (
	"github.com/hashicorp/hcl2/hcl"
)

// File is the top-level object resulting from parsing a configuration file.
type File struct {
	Body  *Body
	Bytes []byte
}

func (f *File) AsHCLFile() *hcl.File {
	return &hcl.File{
		Body:  f.Body,
		Bytes: f.Bytes,

		// TODO: The Nav object, once we have an implementation of it
	}
}
