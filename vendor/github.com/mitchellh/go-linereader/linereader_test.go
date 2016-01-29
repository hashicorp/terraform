package linereader

import (
	"bytes"
	"io"
	"time"
	"reflect"
	"testing"
)

func TestReader(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("foo\nbar\n")

	var result []string
	r := New(&buf)
	for line := range r.Ch {
		result = append(result, line)
	}

	expected := []string{"foo", "bar"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestReader_pause(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte("foo\n"))
		pw.Write([]byte("bar"))
		time.Sleep(200 * time.Millisecond)
		pw.Write([]byte("baz\n"))
	}()

	var result []string
	r := New(pr)
	for line := range r.Ch {
		result = append(result, line)
	}

	expected := []string{"foo", "bar", "baz"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("bad: %#v", result)
	}
}
