package mode

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type sshConfigurable interface {
	agent() bool
	host() string
	port() int
	user() string
	pemFile() string
	hostKey() string
	timeout() time.Duration
	receiveHostKey(string)
}

type sshConfigurator struct {
	provider sshConfigurable
}

func (c *sshConfigurator) sshConfig() (*ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0)
	if c.provider.pemFile() != "" {
		authMethods = append(authMethods, c.publicKeyFile())
	}
	if c.provider.agent() {
		authMethods = append(authMethods, c.sshAgent())
	}

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		c.provider.receiveHostKey(string(ssh.MarshalAuthorizedKey(key)))
		return nil
	}

	if c.provider.hostKey() != "" {
		// from terraform/communicator/ssh/provisioner.go
		// ----------------------------------------------
		// The knownhosts package only takes paths to files, but terraform
		// generally wants to handle config data in-memory. Rather than making
		// the known_hosts file an exception, write out the data to a temporary
		// file to create the HostKeyCallback.
		tf, err := ioutil.TempFile("", "tf-provisioner-known_hosts")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp known_hosts file: %s", err)
		}
		defer tf.Close()
		defer os.RemoveAll(tf.Name())

		// we mark this as a CA as well, but the host key fallback will still
		// use it as a direct match if the remote host doesn't return a
		// certificate.
		if _, err := tf.WriteString(fmt.Sprintf("@cert-authority %s %s\n", c.provider.host(), c.provider.hostKey())); err != nil {
			return nil, fmt.Errorf("failed to write temp known_hosts file: %s", err)
		}
		tf.Sync()

		hostKeyCallback, err = knownhosts.New(tf.Name())
		if err != nil {
			return nil, err
		}
	}

	return &ssh.ClientConfig{
		User:            c.provider.user(),
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.provider.timeout(),
	}, nil
}

func (c *sshConfigurator) sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func (c *sshConfigurator) publicKeyFile() ssh.AuthMethod {
	// public key file is actually not a file, it contains
	// the contents of the file, as documented in
	// - https://www.terraform.io/docs/provisioners/connection.html#private_key
	// = https://www.terraform.io/docs/provisioners/connection.html#bastion_private_key
	// So, don't read the file, just convert it into bytes.
	key, err := ssh.ParsePrivateKey([]byte(c.provider.pemFile()))
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
