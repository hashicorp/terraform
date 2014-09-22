package ssh

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"code.google.com/p/go.crypto/ssh"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
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

// VerifySSH is used to verify the ConnInfo is usable by remote-exec
func VerifySSH(s *terraform.InstanceState) error {
	connType := s.Ephemeral.ConnInfo["type"]
	switch connType {
	case "":
	case "ssh":
	default:
		return fmt.Errorf("Connection type '%s' not supported", connType)
	}
	return nil
}

// ParseSSHConfig is used to convert the ConnInfo of the InstanceState into
// a SSHConfig struct
func ParseSSHConfig(s *terraform.InstanceState) (*SSHConfig, error) {
	sshConf := &SSHConfig{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           sshConf,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.Ephemeral.ConnInfo); err != nil {
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

// PrepareConfig is used to turn the *SSHConfig provided into a
// usable *Config for client initialization.
func PrepareConfig(conf *SSHConfig) (*Config, error) {
	sshConf := &ssh.ClientConfig{
		User: conf.User,
	}
	if conf.KeyFile != "" {
		fullPath, err := homedir.Expand(conf.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to expand home directory: %v", err)
		}
		key, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to read key file '%s': %v", conf.KeyFile, err)
		}

		// We parse the private key on our own first so that we can
		// show a nicer error if the private key has a password.
		block, _ := pem.Decode(key)
		if block == nil {
			return nil, fmt.Errorf(
				"Failed to read key '%s': no key found", conf.KeyFile)
		}
		if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
			return nil, fmt.Errorf(
				"Failed to read key '%s': password protected keys are\n"+
					"not supported. Please decrypt the key prior to use.", conf.KeyFile)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse key file '%s': %v", conf.KeyFile, err)
		}

		sshConf.Auth = append(sshConf.Auth, ssh.PublicKeys(signer))
	}
	if conf.Password != "" {
		sshConf.Auth = append(sshConf.Auth,
			ssh.Password(conf.Password))
		sshConf.Auth = append(sshConf.Auth,
			ssh.KeyboardInteractive(PasswordKeyboardInteractive(conf.Password)))
	}
	host := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	config := &Config{
		SSHConfig:  sshConf,
		Connection: ConnectFunc("tcp", host),
	}
	return config, nil
}
