package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/cli"
)

// VersionCommand is a Command implementation prints the version.
type HelsternwareVersionCommand struct {
	VersionCommand *VersionCommand

	Version           string
	VersionPrerelease string
}

type HelsternwareVersionOutput struct {
	VersionOutput
	HelsternwareVersion string `json:"helsternware_terraform_version"`
}

type BufferedWriterUi struct {
	WriteBuffer bytes.Buffer
	Ui          cli.Ui
}

func (b *BufferedWriterUi) Flush() {
	b.Ui.Output(b.WriteBuffer.String())
}

func (b *BufferedWriterUi) Ask(s string) (string, error) {
	return b.Ui.Ask(s)
}

func (b *BufferedWriterUi) AskSecret(s string) (string, error) {
	return b.Ui.AskSecret(s)
}

func (b *BufferedWriterUi) Output(s string) {
	b.WriteBuffer.WriteString(s)
}

func (b *BufferedWriterUi) Info(s string) {
	b.Ui.Info(s)
}

func (b *BufferedWriterUi) Error(s string) {
	b.Ui.Error(s)
}

func (b *BufferedWriterUi) Warn(s string) {
	b.Ui.Warn(s)
}

func (c *HelsternwareVersionCommand) Help() string {
	return c.VersionCommand.Help()
}

func (c *HelsternwareVersionCommand) Run(args []string) int {

	var bufferedUI = &BufferedWriterUi{
		Ui: c.VersionCommand.Ui,
	}

	c.VersionCommand.Ui = bufferedUI
	var result = c.VersionCommand.Run(args)
	c.VersionCommand.Ui = bufferedUI.Ui

	if result != 0 {
		bufferedUI.Flush()
		return result
	}

	args = c.VersionCommand.Meta.process(args)
	var jsonOutput bool
	cmdFlags := c.VersionCommand.Meta.defaultFlagSet("version")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	_ = cmdFlags.Parse(args)

	var helsternwareVersionString string
	if c.VersionPrerelease != "" {
		helsternwareVersionString = c.Version + "-" + c.VersionPrerelease
	} else {
		helsternwareVersionString = c.Version
	}

	if jsonOutput {
		var jsonBlob = bufferedUI.WriteBuffer.Bytes()
		var output = &HelsternwareVersionOutput{}
		var err = json.Unmarshal(jsonBlob, output)
		if err != nil {
			panic(err)
		}

		output.HelsternwareVersion = helsternwareVersionString

		var jsonOutput []byte
		jsonOutput, err = json.MarshalIndent(output, "", "")
		if err != nil {
			panic(err)
		}

		bufferedUI.Ui.Output(string(jsonOutput))
	} else {
		var versionString bytes.Buffer
		_, _ = fmt.Fprintf(&versionString, "Helsternware Terraform v%s", helsternwareVersionString)
		bufferedUI.Ui.Output(versionString.String())
		bufferedUI.Flush()
	}

	return result
}

func (c *HelsternwareVersionCommand) Synopsis() string {
	return c.VersionCommand.Synopsis()
}
