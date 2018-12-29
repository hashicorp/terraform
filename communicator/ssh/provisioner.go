package ssh

import (
	"bytes"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator/shared"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
	"github.com/xanzy/ssh-agent"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
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
// only keys we look at. If a PrivateKey is given, that is used instead
// of a password.
type connectionInfo struct {
	User       string
	Password   string
	PrivateKey string `mapstructure:"private_key"`
	Host       string
	HostKey    string `mapstructure:"host_key"`
	Port       int
	Agent      bool
	Timeout    string
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`

	BastionUser       string `mapstructure:"bastion_user"`
	BastionPassword   string `mapstructure:"bastion_password"`
	BastionPrivateKey string `mapstructure:"bastion_private_key"`
	BastionHost       string `mapstructure:"bastion_host"`
	BastionHostKey    string `mapstructure:"bastion_host_key"`
	BastionPort       int    `mapstructure:"bastion_port"`

	AgentIdentity string `mapstructure:"agent_identity"`
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

	// Format the host if needed.
	// Needed for IPv6 support.
	connInfo.Host = shared.IpFormat(connInfo.Host)

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
		// Format the bastion host if needed.
		// Needed for IPv6 support.
		connInfo.BastionHost = shared.IpFormat(connInfo.BastionHost)

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

	host := fmt.Sprintf("%s:%d", connInfo.Host, connInfo.Port)

	sshConf, err := buildSSHClientConfig(sshClientConfigOpts{
		user:       connInfo.User,
		host:       host,
		privateKey: connInfo.PrivateKey,
		password:   connInfo.Password,
		hostKey:    connInfo.HostKey,
		sshAgent:   sshAgent,
	})
	if err != nil {
		return nil, err
	}

	connectFunc := ConnectFunc("tcp", host)

	var bastionConf *ssh.ClientConfig
	if connInfo.BastionHost != "" {
		bastionHost := fmt.Sprintf("%s:%d", connInfo.BastionHost, connInfo.BastionPort)

		bastionConf, err = buildSSHClientConfig(sshClientConfigOpts{
			user:       connInfo.BastionUser,
			host:       bastionHost,
			privateKey: connInfo.BastionPrivateKey,
			password:   connInfo.BastionPassword,
			hostKey:    connInfo.HostKey,
			sshAgent:   sshAgent,
		})
		if err != nil {
			return nil, err
		}

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
	host       string
	hostKey    string
}

func buildSSHClientConfig(opts sshClientConfigOpts) (*ssh.ClientConfig, error) {
	hkCallback := ssh.InsecureIgnoreHostKey()

	if opts.hostKey != "" {
		// The knownhosts package only takes paths to files, but terraform
		// generally wants to handle config data in-memory. Rather than making
		// the known_hosts file an exception, write out the data to a temporary
		// file to create the HostKeyCallback.
		tf, err := ioutil.TempFile("", "tf-known_hosts")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp known_hosts file: %s", err)
		}
		defer tf.Close()
		defer os.RemoveAll(tf.Name())

		// we mark this as a CA as well, but the host key fallback will still
		// use it as a direct match if the remote host doesn't return a
		// certificate.
		if _, err := tf.WriteString(fmt.Sprintf("@cert-authority %s %s\n", opts.host, opts.hostKey)); err != nil {
			return nil, fmt.Errorf("failed to write temp known_hosts file: %s", err)
		}
		tf.Sync()

		hkCallback, err = knownhosts.New(tf.Name())
		if err != nil {
			return nil, err
		}
	}

	conf := &ssh.ClientConfig{
		HostKeyCallback: hkCallback,
		User:            opts.user,
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
	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode([]byte(pk))
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	signer, err := ssh.ParsePrivateKey([]byte(pk))
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
		id:    connInfo.AgentIdentity,
	}, nil

}

// A tiny wrapper around an agent.Agent to expose the ability to close its
// associated connection on request.
type sshAgent struct {
	agent agent.Agent
	conn  net.Conn
	id    string
}

func (a *sshAgent) Close() error {
	if a.conn == nil {
		return nil
	}

	return a.conn.Close()
}

// make an attempt to either read the identity file or find a corresponding
// public key file using the typical openssh naming convention.
// This returns the public key in wire format, or nil when a key is not found.
func findIDPublicKey(id string) []byte {
	for _, d := range idKeyData(id) {
		signer, err := ssh.ParsePrivateKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id private key")
			pk := signer.PublicKey()
			return pk.Marshal()
		}

		// try it as a publicKey
		pk, err := ssh.ParsePublicKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id public key")
			return pk.Marshal()
		}

		// finally try it as an authorized key
		pk, _, _, _, err = ssh.ParseAuthorizedKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id authorized key")
			return pk.Marshal()
		}
	}

	return nil
}

// Try to read an id file using the id as the file path. Also read the .pub
// file if it exists, as the id file may be encrypted. Return only the file
// data read. We don't need to know what data came from which path, as we will
// try parsing each as a private key, a public key and an authorized key
// regardless.
func idKeyData(id string) [][]byte {
	idPath, err := filepath.Abs(id)
	if err != nil {
		return nil
	}

	var fileData [][]byte

	paths := []string{idPath}

	if !strings.HasSuffix(idPath, ".pub") {
		paths = append(paths, idPath+".pub")
	}

	for _, p := range paths {
		d, err := ioutil.ReadFile(p)
		if err != nil {
			log.Printf("[DEBUG] error reading %q: %s", p, err)
			continue
		}
		log.Printf("[DEBUG] found identity data at %q", p)
		fileData = append(fileData, d)
	}

	return fileData
}

// sortSigners moves a signer with an agent comment field matching the
// agent_identity to the head of the list when attempting authentication. This
// helps when there are more keys loaded in an agent than the host will allow
// attempts.
func (s *sshAgent) sortSigners(signers []ssh.Signer) {
	if s.id == "" || len(signers) < 2 {
		return
	}

	// if we can locate the public key, either by extracting it from the id or
	// locating the .pub file, then we can more easily determine an exact match
	idPk := findIDPublicKey(s.id)

	// if we have a signer with a connect field that matches the id, send that
	// first, otherwise put close matches at the front of the list.
	head := 0
	for i := range signers {
		pk := signers[i].PublicKey()
		k, ok := pk.(*agent.Key)
		if !ok {
			continue
		}

		// check for an exact match first
		if bytes.Equal(pk.Marshal(), idPk) || s.id == k.Comment {
			signers[0], signers[i] = signers[i], signers[0]
			break
		}

		// no exact match yet, move it to the front if it's close. The agent
		// may have loaded as a full filepath, while the config refers to it by
		// filename only.
		if strings.HasSuffix(k.Comment, s.id) {
			signers[head], signers[i] = signers[i], signers[head]
			head++
			continue
		}
	}

	ss := []string{}
	for _, signer := range signers {
		pk := signer.PublicKey()
		k := pk.(*agent.Key)
		ss = append(ss, k.Comment)
	}
}

func (s *sshAgent) Signers() ([]ssh.Signer, error) {
	signers, err := s.agent.Signers()
	if err != nil {
		return nil, err
	}

	s.sortSigners(signers)
	return signers, nil
}

func (a *sshAgent) Auth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.Signers)
}

func (a *sshAgent) ForwardToAgent(client *ssh.Client) error {
	return agent.ForwardToAgent(client, a.agent)
}
