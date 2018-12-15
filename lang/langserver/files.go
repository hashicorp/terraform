package langserver

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/hashicorp/terraform/tfdiags"
)

type filesystem struct {
	mu   sync.RWMutex
	dirs map[string]*dir
}

type dir struct {
	files map[string]*file
}

type file struct {
	fullPath string
	content  []byte
	open     bool

	// TODO: use a piece table to track edits and flatten into content
	// only when we need to produce a contiguous buffer to parse it.
	// (That'll let us implement the incremental sync mode in the LSP.)

	errs  bool
	diags tfdiags.Diagnostics
	ast   *hcl.File
	wrAST *hclwrite.File
}

func newFilesystem() *filesystem {
	return &filesystem{
		dirs: make(map[string]*dir),
	}
}

func (fs *filesystem) Open(u uri, s []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !u.Valid() {
		return fmt.Errorf("invalid URL to open")
	}

	fullName, dn, fn := u.PathParts()
	d, ok := fs.dirs[dn]
	if !ok {
		d = newDir()
		fs.dirs[dn] = d
	}
	f, ok := d.files[fn]
	if !ok {
		f = newFile(fullName)
	}
	f.content = s
	f.open = true
	d.files[fn] = f
	return nil
}

func (fs *filesystem) Change(u uri, s []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	f.change(s)
	return nil
}

func (fs *filesystem) Close(u uri) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	_, dn, fn := u.PathParts()
	delete(fs.dirs[dn].files, fn)
	return nil
}

func (fs *filesystem) Format(u uri) ([]byte, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(u)
	if f == nil || !f.open {
		return nil, fmt.Errorf("file %q is not open", u)
	}

	s, changed := formatSource(f.content)
	if changed {
		f.change(s)
	}
	return s, nil
}

func (fs *filesystem) FileAST(u uri) *hcl.File {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// FIXME: This method should work with non-open files too, reading them
	// from disk and caching them.
	f := fs.file(u)
	if f == nil {
		return nil
	}

	return f.hclAST()
}

func (fs *filesystem) file(u uri) *file {
	if !u.Valid() {
		return nil
	}
	_, dn, fn := u.PathParts()
	d, ok := fs.dirs[dn]
	if !ok {
		return nil
	}
	return d.files[fn]
}

func newDir() *dir {
	return &dir{
		files: make(map[string]*file),
	}
}

func newFile(fullPath string) *file {
	return &file{fullPath: fullPath}
}

func (f *file) diagnostics() tfdiags.Diagnostics {
	if f.diags != nil {
		return f.diags
	}
	// FIXME: Unfortunate that we just keep re-parsing every time
	// if there are no errors.
	_ = f.hclAST()
	return f.diags
}

func (f *file) hclAST() *hcl.File {
	if f.errs {
		return nil
	}
	if f.ast != nil {
		return f.ast
	}
	hf, diags := hclsyntax.ParseConfig(f.content, f.fullPath, hcl.Pos{Line: 1, Column: 1})
	f.diags = nil
	f.diags = f.diags.Append(diags)
	if diags.HasErrors() {
		f.errs = true
		return nil
	}
	f.ast = hf
	return hf
}

func (f *file) hclWriteAST() *hclwrite.File {
	if f.errs {
		return nil
	}
	if f.wrAST != nil {
		return f.wrAST
	}
	hf, diags := hclwrite.ParseConfig(f.content, f.fullPath, hcl.Pos{Line: 1, Column: 1})
	f.diags = nil
	f.diags = f.diags.Append(diags)
	if diags.HasErrors() {
		f.errs = true
		return nil
	}
	f.wrAST = hf
	return hf
}

func (f *file) change(s []byte) {
	f.content = s
	f.wrAST = nil
	f.ast = nil
	f.diags = nil
	f.errs = false
}
