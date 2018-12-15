package langserver

import (
	"fmt"
	"sync"
)

type filesystem struct {
	mu   sync.RWMutex
	dirs map[string]*dir
}

type dir struct {
	files map[string]*file
}

type file struct {
	content []byte
	open    bool

	// TODO: use a piece table to track edits and flatten into content
	// only when we need to produce a contiguous buffer to parse it.
	// (That'll let us implement the incremental sync mode in the LSP.)
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

	dn, fn := u.DirFilename()
	d, ok := fs.dirs[dn]
	if !ok {
		d = newDir()
		fs.dirs[dn] = d
	}
	f, ok := d.files[fn]
	if !ok {
		f = newFile()
	}
	f.content = s
	f.open = true
	d.files[fn] = f
	return nil
}

func (fs *filesystem) Change(u uri, s []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if !u.Valid() {
		return fmt.Errorf("invalid URL to change")
	}

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

	if !u.Valid() {
		return fmt.Errorf("invalid URL to close")
	}

	dn, fn := u.DirFilename()
	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	delete(fs.dirs[dn].files, fn)
	return nil
}

func (fs *filesystem) file(u uri) *file {
	if !u.Valid() {
		return nil
	}
	dn, fn := u.DirFilename()
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

func newFile() *file {
	return &file{}
}

func (f *file) change(s []byte) {
	f.content = s
}
