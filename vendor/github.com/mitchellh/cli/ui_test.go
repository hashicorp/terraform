package cli

import (
	"bytes"
	"io"
	"testing"
)

func TestBasicUi_implements(t *testing.T) {
	var _ Ui = new(BasicUi)
}

func TestBasicUi_Ask(t *testing.T) {
	in_r, in_w := io.Pipe()
	defer in_r.Close()
	defer in_w.Close()

	writer := new(bytes.Buffer)
	ui := &BasicUi{
		Reader: in_r,
		Writer: writer,
	}

	go in_w.Write([]byte("foo bar\nbaz\n"))

	result, err := ui.Ask("Name?")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if writer.String() != "Name? " {
		t.Fatalf("bad: %#v", writer.String())
	}

	if result != "foo bar" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestBasicUi_AskSecret(t *testing.T) {
	in_r, in_w := io.Pipe()
	defer in_r.Close()
	defer in_w.Close()

	writer := new(bytes.Buffer)
	ui := &BasicUi{
		Reader: in_r,
		Writer: writer,
	}

	go in_w.Write([]byte("foo bar\nbaz\n"))

	result, err := ui.AskSecret("Name?")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if writer.String() != "Name? " {
		t.Fatalf("bad: %#v", writer.String())
	}

	if result != "foo bar" {
		t.Fatalf("bad: %#v", result)
	}
}

func TestBasicUi_Error(t *testing.T) {
	writer := new(bytes.Buffer)
	ui := &BasicUi{Writer: writer}
	ui.Error("HELLO")

	if writer.String() != "HELLO\n" {
		t.Fatalf("bad: %s", writer.String())
	}
}

func TestBasicUi_Error_ErrorWriter(t *testing.T) {
	writer := new(bytes.Buffer)
	ewriter := new(bytes.Buffer)
	ui := &BasicUi{Writer: writer, ErrorWriter: ewriter}
	ui.Error("HELLO")

	if ewriter.String() != "HELLO\n" {
		t.Fatalf("bad: %s", ewriter.String())
	}
}

func TestBasicUi_Output(t *testing.T) {
	writer := new(bytes.Buffer)
	ui := &BasicUi{Writer: writer}
	ui.Output("HELLO")

	if writer.String() != "HELLO\n" {
		t.Fatalf("bad: %s", writer.String())
	}
}

func TestBasicUi_Warn(t *testing.T) {
	writer := new(bytes.Buffer)
	ui := &BasicUi{Writer: writer}
	ui.Warn("HELLO")

	if writer.String() != "HELLO\n" {
		t.Fatalf("bad: %s", writer.String())
	}
}

func TestPrefixedUi_implements(t *testing.T) {
	var _ Ui = new(PrefixedUi)
}

func TestPrefixedUiError(t *testing.T) {
	ui := new(MockUi)
	p := &PrefixedUi{
		ErrorPrefix: "foo",
		Ui:          ui,
	}

	p.Error("bar")
	if ui.ErrorWriter.String() != "foobar\n" {
		t.Fatalf("bad: %s", ui.ErrorWriter.String())
	}
}

func TestPrefixedUiInfo(t *testing.T) {
	ui := new(MockUi)
	p := &PrefixedUi{
		InfoPrefix: "foo",
		Ui:         ui,
	}

	p.Info("bar")
	if ui.OutputWriter.String() != "foobar\n" {
		t.Fatalf("bad: %s", ui.OutputWriter.String())
	}
}

func TestPrefixedUiOutput(t *testing.T) {
	ui := new(MockUi)
	p := &PrefixedUi{
		OutputPrefix: "foo",
		Ui:           ui,
	}

	p.Output("bar")
	if ui.OutputWriter.String() != "foobar\n" {
		t.Fatalf("bad: %s", ui.OutputWriter.String())
	}
}

func TestPrefixedUiWarn(t *testing.T) {
	ui := new(MockUi)
	p := &PrefixedUi{
		WarnPrefix: "foo",
		Ui:         ui,
	}

	p.Warn("bar")
	if ui.ErrorWriter.String() != "foobar\n" {
		t.Fatalf("bad: %s", ui.ErrorWriter.String())
	}
}
