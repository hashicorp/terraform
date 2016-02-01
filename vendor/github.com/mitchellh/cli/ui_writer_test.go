package cli

import (
	"io"
	"testing"
)

func TestUiWriter_impl(t *testing.T) {
	var _ io.Writer = new(UiWriter)
}

func TestUiWriter(t *testing.T) {
	ui := new(MockUi)
	w := &UiWriter{
		Ui: ui,
	}

	w.Write([]byte("foo\n"))
	w.Write([]byte("bar\n"))

	if ui.OutputWriter.String() != "foo\nbar\n" {
		t.Fatalf("bad: %s", ui.OutputWriter.String())
	}
}

func TestUiWriter_empty(t *testing.T) {
	ui := new(MockUi)
	w := &UiWriter{
		Ui: ui,
	}

	w.Write([]byte(""))

	if ui.OutputWriter.String() != "\n" {
		t.Fatalf("bad: %s", ui.OutputWriter.String())
	}
}
