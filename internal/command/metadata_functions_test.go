package command

import (
	"fmt"
	"testing"

	"github.com/mitchellh/cli"
)

func TestMetadataFunctions_error(t *testing.T) {
	ui := new(cli.MockUi)
	c := &MetadataFunctionsCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	if code := c.Run(nil); code != 1 {
		fmt.Println(ui.ErrorWriter.String())
		t.Fatalf("expected error: \n%s", ui.ErrorWriter.String())
	}
}

func TestMetadataFunctions_output(t *testing.T) {
	ui := new(cli.MockUi)
	m := Meta{Ui: ui}
	c := &MetadataFunctionsCommand{Meta: m}

	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
	}

	// gotString := ui.OutputWriter.String()
	// TODO how to mock scope.Functions() to reduce the output length
}
