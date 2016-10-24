package command

import (
	"os"
	"strings"

	"github.com/hashicorp/go-plugin"

	"github.com/hashicorp/nomad/client/driver"
)

type SyslogPluginCommand struct {
	Meta
}

func (e *SyslogPluginCommand) Help() string {
	helpText := `
	This is a command used by Nomad internally to launch a syslog collector"
	`
	return strings.TrimSpace(helpText)
}

func (s *SyslogPluginCommand) Synopsis() string {
	return "internal - lanch a syslog collector plugin"
}

func (s *SyslogPluginCommand) Run(args []string) int {
	if len(args) == 0 {
		s.Ui.Error("log output file isn't provided")
		return 1
	}
	logFileName := args[0]
	stdo, err := os.OpenFile(logFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		s.Ui.Error(err.Error())
		return 1
	}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: driver.HandshakeConfig,
		Plugins:         driver.GetPluginMap(stdo),
	})

	return 0
}
