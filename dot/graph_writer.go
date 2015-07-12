package dot

import (
	"bytes"
	"fmt"
)

// graphWriter wraps a bytes.Buffer and tracks indent level levels.
type graphWriter struct {
	bytes.Buffer
	indent    int
	indentStr string
}

// Returns an initialized graphWriter at indent level 0.
func newGraphWriter() *graphWriter {
	w := &graphWriter{
		indent: 0,
	}
	w.init()
	return w
}

// Prints to the buffer at the current indent level.
func (w *graphWriter) Printf(s string, args ...interface{}) {
	w.WriteString(w.indentStr + fmt.Sprintf(s, args...))
}

// Increase the indent level.
func (w *graphWriter) Indent() {
	w.indent++
	w.init()
}

// Decrease the indent level.
func (w *graphWriter) Unindent() {
	w.indent--
	w.init()
}

func (w *graphWriter) init() {
	indentBuf := new(bytes.Buffer)
	for i := 0; i < w.indent; i++ {
		indentBuf.WriteString("\t")
	}
	w.indentStr = indentBuf.String()
}
