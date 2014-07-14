package remoteexec

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultUser is used if there is no default user given
	DefaultUser = "root"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/script.sh"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = "5m"

	// DefaultShebang is added at the top of the script file
	DefaultShebang = "#!/bin/sh"
)

type ResourceProvisioner struct{}

// SSHConfig is decoded from the ConnInfo of the resource. These
// are the only keys we look at. If a KeyFile is given, that is used
// instead of a password.
type SSHConfig struct {
	User       string
	Password   string
	KeyFile    string `mapstructure:"key_file"`
	Host       string
	Port       int
	Timeout    string
	ScriptPath string `mapstructure:"script_path"`
}

func (p *ResourceProvisioner) Apply(s *terraform.ResourceState,
	c *terraform.ResourceConfig) (*terraform.ResourceState, error) {
	// Ensure the connection type is SSH
	if err := p.verifySSH(s); err != nil {
		return s, err
	}

	// Get the SSH configuration
	_, err := p.sshConfig(s)
	if err != nil {
		return s, err
	}

	// Collect the scripts
	_, err = p.collectScripts(c)
	if err != nil {
		return s, err
	}

	// For-each script, copy + exec
	panic("not implemented")
	return s, nil
}

func (p *ResourceProvisioner) Validate(c *terraform.ResourceConfig) (ws []string, es []error) {
	num := 0
	for name := range c.Raw {
		switch name {
		case "scripts":
			fallthrough
		case "script":
			fallthrough
		case "inline":
			num++
		default:
			es = append(es, fmt.Errorf("Unknown configuration '%s'", name))
		}
	}
	if num != 1 {
		es = append(es, fmt.Errorf("Must provide one of 'scripts', 'script' or 'inline' to remote-exec"))
	}
	return
}

// verifySSH is used to verify the ConnInfo is usable by remote-exec
func (p *ResourceProvisioner) verifySSH(s *terraform.ResourceState) error {
	connType := s.ConnInfo.Raw["type"]
	switch connType {
	case "":
	case "ssh":
	default:
		return fmt.Errorf("Connection type '%s' not supported", connType)
	}
	return nil
}

// sshConfig is used to convert the ConnInfo of the ResourceState into
// a SSHConfig struct
func (p *ResourceProvisioner) sshConfig(s *terraform.ResourceState) (*SSHConfig, error) {
	sshConf := &SSHConfig{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           sshConf,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.ConnInfo.Raw); err != nil {
		return nil, err
	}
	if sshConf.User == "" {
		sshConf.User = DefaultUser
	}
	if sshConf.Port == 0 {
		sshConf.Port = DefaultPort
	}
	if sshConf.ScriptPath == "" {
		sshConf.ScriptPath = DefaultScriptPath
	}
	if sshConf.Timeout == "" {
		sshConf.Timeout = DefaultTimeout
	}
	return sshConf, nil
}

// generateScript takes the configuration and creates a script to be executed
// from the inline configs
func (p *ResourceProvisioner) generateScript(c *terraform.ResourceConfig) (string, error) {
	lines := []string{DefaultShebang}
	command, ok := c.Config["inline"]
	if ok {
		switch cmd := command.(type) {
		case string:
			lines = append(lines, cmd)
		case []string:
			lines = append(lines, cmd...)
		case []interface{}:
			for _, l := range cmd {
				lStr, ok := l.(string)
				if ok {
					lines = append(lines, lStr)
				} else {
					return "", fmt.Errorf("Unsupported 'inline' type! Must be string, or list of strings.")
				}
			}
		default:
			return "", fmt.Errorf("Unsupported 'inline' type! Must be string, or list of strings.")
		}
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n"), nil
}

// collectScripts is used to collect all the scripts we need
// to execute in preperation for copying them.
func (p *ResourceProvisioner) collectScripts(c *terraform.ResourceConfig) ([]io.ReadCloser, error) {
	// Check if inline
	_, ok := c.Config["inline"]
	if ok {
		script, err := p.generateScript(c)
		if err != nil {
			return nil, err
		}
		rc := ioutil.NopCloser(bytes.NewReader([]byte(script)))
		return []io.ReadCloser{rc}, nil
	}

	// Collect scripts
	var scripts []string
	s, ok := c.Config["script"]
	if ok {
		sStr, ok := s.(string)
		if !ok {
			return nil, fmt.Errorf("Unsupported 'script' type! Must be a string.")
		}
		scripts = append(scripts, sStr)
	}

	sl, ok := c.Config["scripts"]
	if ok {
		switch slt := sl.(type) {
		case []string:
			scripts = append(scripts, slt...)
		case []interface{}:
			for _, l := range slt {
				lStr, ok := l.(string)
				if ok {
					scripts = append(scripts, lStr)
				} else {
					return nil, fmt.Errorf("Unsupported 'scripts' type! Must be list of strings.")
				}
			}
		default:
			return nil, fmt.Errorf("Unsupported 'scripts' type! Must be list of strings.")
		}
	}

	// Open all the scripts
	var fhs []io.ReadCloser
	for _, s := range scripts {
		fh, err := os.Open(s)
		if err != nil {
			for _, fh := range fhs {
				fh.Close()
			}
			return nil, fmt.Errorf("Failed to open script '%s': %v", s, err)
		}
		fhs = append(fhs, fh)
	}

	// Done, return the file handles
	return fhs, nil
}
