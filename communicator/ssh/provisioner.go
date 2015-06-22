package ssh

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
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
	KeyFile    string `mapstructure:"key_file"`
	Host       string
	Port       int
	Agent      bool
	Timeout    string
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`

	BastionUser     string `mapstructure:"bastion_user"`
	BastionPassword string `mapstructure:"bastion_password"`
	BastionKeyFile  string `mapstructure:"bastion_key_file"`
	BastionHost     string `mapstructure:"bastion_host"`
	BastionPort     int    `mapstructure:"bastion_port"`
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

	// Default all bastion config attrs to their non-bastion counterparts
	if connInfo.BastionHost != "" {
		if connInfo.BastionUser == "" {
			connInfo.BastionUser = connInfo.User
		}
		if connInfo.BastionPassword == "" {
			connInfo.BastionPassword = connInfo.Password
		}
		if connInfo.BastionKeyFile == "" {
			connInfo.BastionKeyFile = connInfo.KeyFile
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
		user:     connInfo.User,
		keyFile:  connInfo.KeyFile,
		password: connInfo.Password,
		sshAgent: sshAgent,
	})
	if err != nil {
		return nil, err
	}

	var bastionConf *ssh.ClientConfig
	if connInfo.BastionHost != "" {
		bastionConf, err = buildSSHClientConfig(sshClientConfigOpts{
			user:     connInfo.BastionUser,
			keyFile:  connInfo.BastionKeyFile,
			password: connInfo.BastionPassword,
			sshAgent: sshAgent,
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
	keyFile  string
	password string
	sshAgent *sshAgent
	user     string
}

func buildSSHClientConfig(opts sshClientConfigOpts) (*ssh.ClientConfig, error) {
	conf := &ssh.ClientConfig{
		User: opts.user,
	}

	if opts.sshAgent != nil {
		conf.Auth = append(conf.Auth, opts.sshAgent.Auth())
	}

	if opts.keyFile != "" {
		pubKeyAuth, err := readPublicKeyFromPath(opts.keyFile)
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

	return conf, nil
}

func readPublicKeyFromPath(path string) (ssh.AuthMethod, error) {
	fullPath, err := homedir.Expand(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to expand home directory: %s", err)
	}
	key, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read key file %q: %s", path, err)
	}

	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode(key)
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", path)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", path)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse key file %q: %s", path, err)
	}

	return ssh.PublicKeys(signer), nil
}

func connectToAgent(connInfo *connectionInfo) (*sshAgent, error) {
	if connInfo.Agent != true {
		// No agent configured
		return nil, nil
	}

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")

	if sshAuthSock == "" {
		return nil, fmt.Errorf("SSH Requested but SSH_AUTH_SOCK not-specified")
	}

	conn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to SSH_AUTH_SOCK: %v", err)
	}

	// connection close is handled over in Communicator
	return &sshAgent{
		agent: agent.NewClient(conn),
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
	return a.conn.Close()
}

func (a *sshAgent) Auth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.agent.Signers)
}

func (a *sshAgent) ForwardToAgent(client *ssh.Client) error {
	return agent.ForwardToAgent(client, a.agent)
}
