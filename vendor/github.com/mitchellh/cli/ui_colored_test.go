package cli

import (
	"testing"
)

func TestColoredUi_impl(t *testing.T) {
	var _ Ui = new(ColoredUi)
}

func TestColoredUi_noColor(t *testing.T) {
	mock := new(MockUi)
	ui := &ColoredUi{
		ErrorColor: UiColorNone,
		Ui:         mock,
	}
	ui.Error("foo")

	if mock.ErrorWriter.String() != "foo\n" {
		t.Fatalf("bad: %#v", mock.ErrorWriter.String())
	}
}

func TestColoredUi_Error(t *testing.T) {
	mock := new(MockUi)
	ui := &ColoredUi{
		ErrorColor: UiColor{Code: 33},
		Ui:         mock,
	}
	ui.Error("foo")

	if mock.ErrorWriter.String() != "\033[0;33mfoo\033[0m\n" {
		t.Fatalf("bad: %#v", mock.ErrorWriter.String())
	}
}

func TestColoredUi_Info(t *testing.T) {
	mock := new(MockUi)
	ui := &ColoredUi{
		InfoColor: UiColor{Code: 33},
		Ui:        mock,
	}
	ui.Info("foo")

	if mock.OutputWriter.String() != "\033[0;33mfoo\033[0m\n" {
		t.Fatalf("bad: %#v %#v", mock.OutputWriter.String())
	}
}

func TestColoredUi_Output(t *testing.T) {
	mock := new(MockUi)
	ui := &ColoredUi{
		OutputColor: UiColor{Code: 33},
		Ui:          mock,
	}
	ui.Output("foo")

	if mock.OutputWriter.String() != "\033[0;33mfoo\033[0m\n" {
		t.Fatalf("bad: %#v %#v", mock.OutputWriter.String())
	}
}

func TestColoredUi_Warn(t *testing.T) {
	mock := new(MockUi)
	ui := &ColoredUi{
		WarnColor: UiColor{Code: 33},
		Ui:        mock,
	}
	ui.Warn("foo")

	if mock.ErrorWriter.String() != "\033[0;33mfoo\033[0m\n" {
		t.Fatalf("bad: %#v %#v", mock.ErrorWriter.String())
	}
}
