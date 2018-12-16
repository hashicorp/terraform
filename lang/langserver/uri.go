package langserver

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/internal/lsp"
)

const uriPrefix = "file://"

type uri string

func (u uri) Valid() bool {
	if !strings.HasPrefix(string(u), uriPrefix) {
		return false
	}
	p := string(u[len(uriPrefix):])
	_, err := url.PathUnescape(p)
	if err != nil {
		return false
	}
	return true
}

func (u uri) FullPath() string {
	if !u.Valid() {
		panic("invalid uri")
	}
	p := string(u[len(uriPrefix):])
	p, _ = url.PathUnescape(p)
	return filepath.FromSlash(p)
}

func (u uri) Dir() string {
	return filepath.Dir(u.FullPath())
}

func (u uri) Filename() string {
	return filepath.Base(u.FullPath())
}

func (u uri) PathParts() (full, dir, filename string) {
	full = u.FullPath()
	dir = filepath.Dir(full)
	filename = filepath.Base(full)
	return full, dir, filename
}

func (u uri) LSPDocumentURI() lsp.DocumentURI {
	return lsp.DocumentURI(u)
}
