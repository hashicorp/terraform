package ssh

import (
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/ssh-agent"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "root"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/terraform_%RAND%.sh"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute
)

// connectionInfo is decoded from the ConnInfo of the resource. These are the
// only keys we look at. If a KeyFile is given, that is used instead
// of a password.
type connectionInfo struct {
	User       string
	Password   string
	PrivateKey string `mapstructure:"private_key"`
	Host       string
	Port       int
	Agent      bool
	Timeout    string
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`

	BastionUser       string `mapstructure:"bastion_user"`
	BastionPassword   string `mapstructure:"bastion_password"`
	BastionPrivateKey string `mapstructure:"bastion_private_key"`
	BastionHost       string `mapstructure:"bastion_host"`
	BastionPort       int    `mapstructure:"bastion_port"`

	// Deprecated
	KeyFile        string `mapstructure:"key_file"`
	BastionKeyFile string `mapstructure:"bastion_key_file"`
}

// parseConnectionInfo is used to convert the ConnInfo of the InstanceState into
// a ConnectionInfo struct
func parseConnectionInfo(s *terraform.InstanceState) (*connectionInfo, error) {
	connInfo := &connectionInfo{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           connInfo,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.Ephemeral.ConnInfo); err != nil {
		return nil, err
	}

	// To default Agent to true, we need to check the raw string, since the
	// decoded boolean can't represent "absence of config".
	//
	// And if SSH_AUTH_SOCK is not set, there's no agent to connect to, so we
	// shouldn't try.
	if s.Ephemeral.ConnInfo["agent"] == "" && os.Getenv("SSH_AUTH_SOCK") != "" {
		connInfo.Agent = true
	}

	if connInfo.User == "" {
		connInfo.User = DefaultUser
	}
	if connInfo.Port == 0 {
		connInfo.Port = DefaultPort
	}
	if connInfo.ScriptPath == "" {
		connInfo.ScriptPath = DefaultScriptPath
	}
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
	}

	// Load deprecated fields; we can handle either path or contents in
	// underlying implementation.
	if connInfo.PrivateKey == "" && connInfo.KeyFile != "" {
		connInfo.PrivateKey = connInfo.KeyFile
	}
	if connInfo.BastionPrivateKey == "" && connInfo.BastionKeyFile != "" {
		connInfo.BastionPrivateKey = connInfo.BastionKeyFile
	}

	// Default all bastion config attrs to their non-bastion counterparts
	if connInfo.BastionHost != "" {
		if connInfo.BastionUser == "" {
			connInfo.BastionUser = connInfo.User
		}
		if connInfo.BastionPassword == "" {
			connInfo.BastionPassword = connInfo.Password
		}
		if connInfo.BastionPrivateKey == "" {
			connInfo.BastionPrivateKey = connInfo.PrivateKey
		}
		if connInfo.BastionPort == 0 {
			connInfo.BastionPort = connInfo.Port
		}
	}

	return connInfo, nil
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

// prepareSSHConfig is used to turn the *ConnectionInfo provided into a
// usable *SSHConfig for client initialization.
func prepareSSHConfig(connInfo *connectionInfo) (*sshConfig, error) {
	sshAgent, err := connectToAgent(connInfo)
	if err != nil {
		return nil, err
	}

	sshConf, err := buildSSHClientConfig(sshClientConfigOpts{
		user:       connInfo.User,
		privateKey: connInfo.PrivateKey,
		password:   connInfo.Password,
		sshAgent:   sshAgent,
	})
	if err != nil {
		return nil, err
	}

	var bastionConf *ssh.ClientConfig
	if connInfo.BastionHost != "" {
		bastionConf, err = buildSSHClientConfig(sshClientConfigOpts{
			user:       connInfo.BastionUser,
			privateKey: connInfo.BastionPrivateKey,
			password:   connInfo.BastionPassword,
			sshAgent:   sshAgent,
		})
		if err != nil {
			return nil, err
		}
	}

	host := fmt.Sprintf("%s:%d", connInfo.Host, connInfo.Port)
	connectFunc := ConnectFunc("tcp", host)

	if bastionConf != nil {
		bastionHost := fmt.Sprintf("%s:%d", connInfo.BastionHost, connInfo.BastionPort)
		connectFunc = BastionConnectFunc("tcp", bastionHost, bastionConf, "tcp", host)
	}

	config := &sshConfig{
		config:     sshConf,
		connection: connectFunc,
		sshAgent:   sshAgent,
	}
	return config, nil
}

type sshClientConfigOpts struct {
	privateKey string
	password   string
	sshAgent   *sshAgent
	user       string
}

func buildSSHClientConfig(opts sshClientConfigOpts) (*ssh.ClientConfig, error) {
	conf := &ssh.ClientConfig{
		User: opts.user,
	}

	if opts.privateKey != "" {
		pubKeyAuth, err := readPrivateKey(opts.privateKey)
		if err != nil {
			return nil, err
		}
		conf.Auth = append(conf.Auth, pubKeyAuth)
	}

	if opts.password != "" {
		conf.Auth = append(conf.Auth, ssh.Password(opts.password))
		conf.Auth = append(conf.Auth, ssh.KeyboardInteractive(
			PasswordKeyboardInteractive(opts.password)))
	}

	if opts.sshAgent != nil {
		conf.Auth = append(conf.Auth, opts.sshAgent.Auth())
	}

	return conf, nil
}

func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	key, _, err := pathorcontents.Read(pk)
	if err != nil {
		return nil, fmt.Errorf("Failed to read private key %q: %s", pk, err)
	}

	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key file %q: %s", pk, err)
	}

	return ssh.PublicKeys(signer), nil
}

func connectToAgent(connInfo *connectionInfo) (*sshAgent, error) {
	if connInfo.Agent != true {
		// No agent configured
		return nil, nil
	}

	agent, conn, err := sshagent.New()
	if err != nil {
		return nil, err
	}

	// connection close is handled over in Communicator
	return &sshAgent{
		agent: agent,
		conn:  conn,
	}, nil

}

// A tiny wrapper around an agent.Agent to expose the ability to close its
// associated connection on request.
type sshAgent struct {
	agent agent.Agent
	conn  net.Conn
}

func (a *sshAgent) Close() error {
	if a.conn == nil {
		return nil
	}

	return a.conn.Close()
}

func (a *sshAgent) Auth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.agent.Signers)
}

func (a *sshAgent) ForwardToAgent(client *ssh.Client) error {
	return agent.ForwardToAgent(client, a.agent)
}
