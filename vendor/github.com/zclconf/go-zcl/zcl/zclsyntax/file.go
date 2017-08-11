package zclsyntax

import "github.com/zclconf/go-zcl/zcl"

// File is the top-level object resulting from parsing a configuration file.
type File struct {
	Body  *Body
	Bytes []byte
}

func (f *File) AsZCLFile() *zcl.File {
	return &zcl.File{
		Body:  f.Body,
		Bytes: f.Bytes,

		// TODO: The Nav object, once we have an implementation of it
	}
}
