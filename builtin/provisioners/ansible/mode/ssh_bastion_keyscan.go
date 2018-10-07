package mode

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/terraform"
	linereader "github.com/mitchellh/go-linereader"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh"
)

const (
	homeSSHDirectory = "~/.ssh"
)

type cleanup func()

type bastionKeyScan struct {
	o                 terraform.UIOutput
	sshClient         *ssh.Client
	host              string
	port              int
	sshKeyscanTimeout int
}

func newBastionKeyScan(o terraform.UIOutput,
	sshClient *ssh.Client,
	host string,
	port int,
	sshKeyscanTimeout int) *bastionKeyScan {
	return &bastionKeyScan{
		o:                 o,
		sshClient:         sshClient,
		host:              host,
		port:              port,
		sshKeyscanTimeout: sshKeyscanTimeout,
	}
}

func (b *bastionKeyScan) sshModes() ssh.TerminalModes {
	return ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
}

func (b *bastionKeyScan) makeError(pattern string, e error) error {
	if e == nil {
		return fmt.Errorf("Bastion ssh-keyscan: %s", pattern)
	}
	return fmt.Errorf("Bastion ssh-keyscan: %s", fmt.Sprintf(pattern, e))
}

func (b *bastionKeyScan) output(message string) {
	b.o.Output(fmt.Sprintf("Bastion host: %s", message))
}

func (b *bastionKeyScan) copyOutput(r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		b.o.Output(line)
	}
}

func (b *bastionKeyScan) redirectOutputs(s *ssh.Session) (cleanup, error) {
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go b.copyOutput(outR, outDoneCh)
	go b.copyOutput(errR, errDoneCh)
	stdout, err := s.StdoutPipe()

	cleanupF := func() {
		outW.Close()
		errW.Close()
		<-outDoneCh
		<-errDoneCh
	}

	if err != nil {
		cleanupF()
		return nil, fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go io.Copy(outW, stdout)

	stderr, err := s.StderrPipe()
	if err != nil {
		cleanupF()
		return nil, fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go io.Copy(errW, stderr)

	return cleanupF, nil
}

func (b *bastionKeyScan) execute(command string) error {
	b.output(fmt.Sprintf("running command: %s", command))
	session, err := b.sshClient.NewSession()
	if err != nil {
		return b.makeError("failed to create session: %s.", err)
	}
	defer session.Close()
	if err := session.RequestPty("xterm", 80, 40, b.sshModes()); err != nil {
		return b.makeError("request for pseudo terminal failed: %s.", err)
	}
	cleanupF, err := b.redirectOutputs(session)
	if err != nil {
		return err
	}
	defer cleanupF()
	commandResult := session.Run(command)
	return commandResult
}

func (b *bastionKeyScan) scan() (string, error) {

	b.output(fmt.Sprintf("ensuring the existence of '%s'...", homeSSHDirectory))
	if err := b.execute(
		fmt.Sprintf(
			"mkdir -p \"%s\"",
			b.quotedSSHKnownFileDir())); err != nil {
		return "", err
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(b.quotedSSHKnownFileDir(), u1.String())

	timeoutMs := b.sshKeyscanTimeout * 1000
	timeSpentMs := 0
	intervalMs := 5000

	sshKeyScanCommand := fmt.Sprintf("ssh_keyscan_result=$(ssh-keyscan -T %d -p %d %s 2>/dev/null | grep %s) && echo -e \"${ssh_keyscan_result}\" > \"%s\"",
		b.sshKeyscanTimeout,
		b.port,
		b.host,
		b.host,
		targetPath)

	// do not rely just on the ssh-keyscan -T;
	// it may take time until the instance starts replying to ssh requests
	// until then, we may be getting "no route to host",
	// in such case the keyscan would fail regardless of timeout
	// we need to repeat until we succeed or time out
	for {
		keyScanError := b.execute(sshKeyScanCommand)
		if keyScanError == nil {
			break
		}
		b.output(fmt.Sprintf("ssh-keyscan hasn't succeeded yet (last error: %s); retrying...", keyScanError))
		time.Sleep(time.Duration(intervalMs) * time.Millisecond)
		timeSpentMs = timeSpentMs + intervalMs
		if timeSpentMs > timeoutMs {
			return "", b.makeError(
				fmt.Sprintf(
					"failed receive target ssh key for %s:%d within time specified period of %d seconds.",
					b.host, b.port, b.sshKeyscanTimeout), nil)
		}
	}

	// read and remove the temporary known hosts file:
	var buf bytes.Buffer
	session, err := b.sshClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	session.Stdout = &buf
	if err := session.Run(fmt.Sprintf("echo -e \"$(cat \"%s\")\"", targetPath)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (b *bastionKeyScan) quotedSSHKnownFileDir() string {
	return strings.Replace(homeSSHDirectory, "~/", "$HOME/", 1)
}
