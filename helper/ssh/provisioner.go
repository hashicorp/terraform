package ssh

import (
	"fmt"
	"log"
	"time"

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
	DefaultTimeout = 5 * time.Minute
)

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
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`
}

// verifySSH is used to verify the ConnInfo is usable by remote-exec
func VerifySSH(s *terraform.ResourceState) error {
	connType := s.ConnInfo["type"]
	switch connType {
	case "":
	case "ssh":
	default:
		return fmt.Errorf("Connection type '%s' not supported", connType)
	}
	return nil
}

// ParseSSHConfig is used to convert the ConnInfo of the ResourceState into
// a SSHConfig struct
func ParseSSHConfig(s *terraform.ResourceState) (*SSHConfig, error) {
	sshConf := &SSHConfig{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           sshConf,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.ConnInfo); err != nil {
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
	if sshConf.Timeout != "" {
		sshConf.TimeoutVal = safeDuration(sshConf.Timeout, DefaultTimeout)
	} else {
		sshConf.TimeoutVal = DefaultTimeout
	}
	return sshConf, nil
}

// safeDuration returns either the parsed duration or a default value
func safeDuration(dur string, defaultDur time.Duration) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		log.Printf("Invalid duration '%s', using default of %s", dur, defaultDur)
		return defaultDur
	}
	return d
}
