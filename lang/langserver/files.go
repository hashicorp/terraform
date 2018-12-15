package langserver

import (
	"fmt"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	encunicode "golang.org/x/text/encoding/unicode"

	"github.com/hashicorp/terraform/internal/lsp"
	"github.com/hashicorp/terraform/tfdiags"
)

var utf16encoding = encunicode.UTF16(encunicode.LittleEndian, encunicode.IgnoreBOM)
var utf16encoder = utf16encoding.NewEncoder()
var utf16decoder = utf16encoding.NewDecoder()

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

	ls    sourceLines
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

func (fs *filesystem) Change(u uri, changes []lsp.TextDocumentContentChangeEvent) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f := fs.file(u)
	if f == nil || !f.open {
		return fmt.Errorf("file %q is not open", u)
	}
	for _, change := range changes {
		f.applyChange(change)
	}
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

	s, _ := formatSource(f.content)
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

func (fs *filesystem) FileDiagnostics(u uri) tfdiags.Diagnostics {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// FIXME: This method should work with non-open files too, reading them
	// from disk and caching them.
	f := fs.file(u)
	if f == nil {
		return nil
	}

	return f.diagnostics()
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

func (f *file) lines() sourceLines {
	if f.ls == nil {
		f.ls = makeSourceLines(f.fullPath, f.content)
	}
	return f.ls
}

func (f *file) applyChange(ch lsp.TextDocumentContentChangeEvent) {
	// Change positions/lengths are described in UTF-16 code units relative
	// to the start of a line, so to ensure we apply exactly what the client
	// is requesting (including weird conditions like typing into the middle
	// of a UTF-16 surrogate pair) we will transcode to UTF-16, apply the
	// edit, and transcode back. This can potentially cause a lot of churn
	// of large buffers, so we may wish to optimize this more in future, but
	// at least for now we'll limit the window of the buffer that we convert
	// to UTF-16.

	ls := f.lines()
	startLine := int(ch.Range.Start.Line)
	endLine := int(ch.Range.End.Line)
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(ls) {
		endLine = len(ls) - 1
	}
	startChar := int(ch.Range.Start.Character)
	endChar := int(ch.Range.End.Character)

	startByte := ls[startLine].rng.Start.Byte
	endByte := ls[endLine].rng.End.Byte
	lastLineStartByte := ls[endLine].rng.Start.Byte
	// We take some care to avoid panics here but none of these situations
	// should actually arise for a well-behaved client.
	if lastLineStartByte > endByte {
		lastLineStartByte = endByte
	}
	if startByte > lastLineStartByte {
		startByte = lastLineStartByte
	}
	if startByte < 0 {
		startByte = 0
	}
	if endByte > len(f.content) {
		endByte = len(f.content) - 1
	}
	if lastLineStartByte > len(f.content) {
		lastLineStartByte = len(f.content) - 1
	}

	inU8buf := f.content[startByte:endByte]
	// We need to figure out now where in the UTF-16 buffer our lastLineStartByte
	// will end up, so we can properly slice using our end position's character.
	lastLineStartByteU16 := 0
	for b := inU8buf[:lastLineStartByte-startByte]; len(b) > 0; {
		r, l := utf8.DecodeRune(b)
		b = b[l:]
		if r1, r2 := utf16.EncodeRune(r); r1 == 0xfffd && r2 == 0xfffd {
			lastLineStartByteU16 += 2 // encoded as one 16-bit unit
		} else {
			lastLineStartByteU16 += 4 // encoded as two 16-bit units
		}
	}

	inU16buf, err := utf16encoder.Bytes(inU8buf)
	if err != nil {
		// Should never happen since errors are handled by inserting marker characters
		panic("utf16encoder failed")
	}

	replU16buf, err := utf16encoder.Bytes([]byte(ch.Text))
	if err != nil {
		panic("utf16encoder failed")
	}

	outU16BufLen := len(inU16buf) - (int(ch.RangeLength) * 2) + len(replU16buf)
	outU16Buf := make([]byte, 0, outU16BufLen)
	outU16Buf = append(outU16Buf, inU16buf[:startChar*2]...)
	outU16Buf = append(outU16Buf, replU16buf...)
	outU16Buf = append(outU16Buf, inU16buf[lastLineStartByteU16+endChar*2:]...)

	outU8Buf, err := utf16decoder.Bytes(outU16Buf)
	if err != nil {
		panic("utf16decoder failed")
	}

	var resultBuf []byte
	resultBuf = append(resultBuf, f.content[:startByte]...)
	resultBuf = append(resultBuf, outU8Buf...)
	resultBuf = append(resultBuf, f.content[endByte:]...)

	f.change(resultBuf)
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
	f.ls = nil
	f.wrAST = nil
	f.ast = nil
	f.diags = nil
	f.errs = false
}
