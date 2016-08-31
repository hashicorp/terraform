package bytesnoerror

import (
	"bytes"
	"fmt"
	"io"
)

type Buffer struct {
	buf bytes.Buffer
}

func (b *Buffer) WriteString(s string) {
	_, err := b.buf.WriteString(s)
	if err != nil {
		panic(err) // write to bytes.Buffer never fails
	}
}

func (b *Buffer) WriteBytes(p []byte) {
	_, err := b.buf.Write(p)
	if err != nil {
		panic(err) // write to bytes.Buffer never fails
	}
}

func (b *Buffer) WriteRune(r rune) {
	_, err := b.buf.WriteRune(r)
	if err != nil {
		panic(err) // write to bytes.Buffer never fails
	}
}

func (b *Buffer) Writef(format string, a ...interface{}) {
	_, err := fmt.Fprintf(&b.buf, format, a...)
	if err != nil {
		panic(err) // write to bytes.Buffer never fails
	}
}

func (b *Buffer) Writeln(a ...interface{}) {
	_, err := fmt.Fprintln(&b.buf, a...)
	if err != nil {
		panic(err) // write to bytes.Buffer never fails
	}
}

func (b *Buffer) Len() int {
	return b.buf.Len()
}

func (b *Buffer) String() string {
	return b.buf.String()
}

func (b *Buffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *Buffer) Truncate(n int) {
	b.buf.Truncate(n)
}

// Must return error to implement io.Writer.
func (b *Buffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

// Must return error to implement io.Reader.
func (b *Buffer) Read(p []byte) (n int, err error) {
	return b.buf.Read(p)
}

// Must return error because the Reader might fail.
func (b *Buffer) ReadFrom(r io.Reader) (n int64, err error) {
	return b.buf.ReadFrom(r)
}
