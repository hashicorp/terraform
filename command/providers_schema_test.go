package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestProvidersSchema_error(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ProvidersSchemaCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run(nil); code != 0 {
		t.Fatalf("bad: \n%s", ui.OutputWriter.String())
	}
}
