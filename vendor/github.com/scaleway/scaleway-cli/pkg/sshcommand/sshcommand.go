package sshcommand

import (
	"fmt"
	"runtime"
	"strings"
)

// Command contains settings to build a ssh command
type Command struct {
	Host                   string
	User                   string
	Port                   int
	SSHOptions             []string
	Gateway                *Command
	Command                []string
	Debug                  bool
	NoEscapeCommand        bool
	SkipHostKeyChecking    bool
	Quiet                  bool
	AllocateTTY            bool
	EnableSSHKeyForwarding bool

	isGateway bool
}

// New returns a minimal Command
func New(host string) *Command {
	return &Command{
		Host: host,
	}
}

func (c *Command) applyDefaults() {
	if strings.Contains(c.Host, "@") {
		parts := strings.Split(c.Host, "@")
		c.User = parts[0]
		c.Host = parts[1]
	}

	if c.Port == 0 {
		c.Port = 22
	}

	if c.isGateway {
		c.SSHOptions = []string{"-W", "%h:%p"}
	}
}

// Slice returns an execve compatible slice of arguments
func (c *Command) Slice() []string {
	c.applyDefaults()

	slice := []string{}

	slice = append(slice, "ssh")

	if c.EnableSSHKeyForwarding {
		slice = append(slice, "-A")
	}

	if c.Quiet {
		slice = append(slice, "-q")
	}

	if c.SkipHostKeyChecking {
		slice = append(slice, "-o", "UserKnownHostsFile=/dev/null", "-o", "StrictHostKeyChecking=no")
	}

	if len(c.SSHOptions) > 0 {
		slice = append(slice, c.SSHOptions...)
	}

	if c.Gateway != nil {
		c.Gateway.isGateway = true
		slice = append(slice, "-o", "ProxyCommand="+c.Gateway.String())
	}

	if c.User != "" {
		slice = append(slice, "-l", c.User)
	}

	slice = append(slice, c.Host)

	if c.AllocateTTY {
		slice = append(slice, "-t", "-t")
	}

	slice = append(slice, "-p", fmt.Sprintf("%d", c.Port))
	if len(c.Command) > 0 {
		slice = append(slice, "--", "/bin/sh", "-e")
		if c.Debug {
			slice = append(slice, "-x")
		}
		slice = append(slice, "-c")

		var escapedCommand []string
		if c.NoEscapeCommand {
			escapedCommand = c.Command
		} else {
			escapedCommand = []string{}
			for _, part := range c.Command {
				escapedCommand = append(escapedCommand, fmt.Sprintf("%q", part))
			}
		}
		slice = append(slice, fmt.Sprintf("%q", strings.Join(escapedCommand, " ")))
	}
	if runtime.GOOS == "windows" {
		slice[len(slice)-1] = slice[len(slice)-1] + " " // Why ?
	}
	return slice
}

// String returns a copy-pasteable command, useful for debugging
func (c *Command) String() string {
	slice := c.Slice()
	for i := range slice {
		quoted := fmt.Sprintf("%q", slice[i])
		if strings.Contains(slice[i], " ") || len(quoted) != len(slice[i])+2 {
			slice[i] = quoted
		}
	}
	return strings.Join(slice, " ")
}
