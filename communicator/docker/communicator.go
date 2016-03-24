package docker

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bufio"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	// DefaultShebang is added at the top of a inline converted script file
	DefaultShebang = "#!/bin/sh\n"
)

// Communicator represents the Docker communicator
type Communicator struct {
	connInfo *connectionInfo
	client   *dc.Client
}

func New(s *terraform.InstanceState) (*Communicator, error) {

	log.Printf("Creating new docker communicator for instance state %s", s.String())
	connInfo, err := parseConnectionInfo(s)
	if err != nil {
		return nil, err
	}
	var client *dc.Client
	// If there is no cert information, then just return the direct client
	if connInfo.CertPath == "" {
		client, err = dc.NewClient(connInfo.Host)
		if err != nil {
			return nil, err
		}
	} else {

		// If there is cert information, load it and use it.
		ca := filepath.Join(connInfo.CertPath, "ca.pem")
		cert := filepath.Join(connInfo.CertPath, "cert.pem")
		key := filepath.Join(connInfo.CertPath, "key.pem")
		client, err = dc.NewTLSClient(connInfo.Host, cert, key, ca)
		if err != nil {
			return nil, err
		}
	}
	comm := &Communicator{
		connInfo: connInfo,
		client:   client,
	}
	return comm, nil
}

// Connect is used to setup the connection
func (c *Communicator) Connect(o terraform.UIOutput) error {
	err := c.client.Ping()
	if err != nil {
		return fmt.Errorf("Error pinging Docker server: %s", err)
	}
	return nil
}

// Disconnect is used to terminate the connection
func (c *Communicator) Disconnect() error {
	return nil
}

// Timeout returns the configured connection timeout
func (c *Communicator) Timeout() time.Duration {
	return c.connInfo.TimeoutVal
}

// ScriptPath returns the configured script path
func (c *Communicator) ScriptPath() string {
	log.Printf("Generating script path from connection info %s", c.connInfo.ScriptPath)
	path := strings.Replace(
		c.connInfo.ScriptPath, "%RAND%",
		strconv.FormatInt(int64(rand.Int31()), 10), -1)
	log.Printf("Generated script path: %s", path)
	return path
}

// Start executes a remote command in a new session
func (c *Communicator) Start(cmd *remote.Cmd) error {
	log.Printf("Starting new docker command.")
	createOptions := dc.CreateExecOptions{
		AttachStdin:  false,
		AttachStderr: true,
		AttachStdout: true,
		Container:    c.connInfo.ContainerId,
		Cmd:          strings.Fields(cmd.Command),
		Tty:          true,
	}
	var exec *dc.Exec
	var err error
	if exec, err = c.client.CreateExec(createOptions); err != nil {
		return err
	}
	log.Printf("Execution planned with id %q", exec.ID)
	log.Printf("starting remote command: %s", cmd.Command)
	startOptions := dc.StartExecOptions{
		Detach:       false,
		Tty:          true,
		InputStream:  cmd.Stdin,
		OutputStream: cmd.Stdout,
		ErrorStream:  cmd.Stderr,
		// If RawTerminal is set to false, then Tty should be set to false in both
		// create and start exec otherwise it results in a docker 'Unrecognized input header' error
		RawTerminal: true,
	}
	if err = c.client.StartExec(exec.ID, startOptions); err != nil {
		return err
	}
	log.Printf("Retriving exit code for Execution with id %s", exec.ID)
	var inspect *dc.ExecInspect
	if inspect, err = c.client.InspectExec(exec.ID); err != nil {
		return err
	}
	log.Printf("Command return code was: %q", inspect.ExitCode)
	cmd.SetExited(inspect.ExitCode)
	return nil
}

// Upload is used to upload a single file
func (c *Communicator) Upload(path string, input io.Reader) error {
	log.Printf("Uploading to container %s", path)
	var tarball io.Reader
	var err error
	if tarball, err = generateSingleFileTar(path, input); err != nil {
		return err
	}

	return c.uploadTar(filepath.Dir(path), tarball)
}

// UploadScript is used to upload a file as a executable script
func (c *Communicator) UploadScript(path string, input io.Reader) error {
	log.Printf("Uploading script to container %s", path)
	reader := bufio.NewReader(input)
	prefix, err := reader.Peek(2)
	if err != nil {
		return fmt.Errorf("Error reading script: %s", err)
	}

	var script bytes.Buffer
	if string(prefix) != "#!" {
		script.WriteString(DefaultShebang)
	}

	script.ReadFrom(reader)
	return c.Upload(path, &script)
}

// UploadDir is used to upload a directory
func (c *Communicator) UploadDir(dst string, src string) error {
	log.Printf("Uploading directory %s to %s in the container", src, dst)
	var tarball io.Reader
	var err error
	if tarball, err = generateDirTar(src); err != nil {
		return err
	}
	return c.uploadTar(dst, tarball)
}

func (c *Communicator) uploadTar(path string, tar io.Reader) error {
	log.Printf("Uploading to container %s", path)
	return c.client.UploadToContainer(c.connInfo.ContainerId, dc.UploadToContainerOptions{InputStream: tar, Path: path})
}

func generateDirTar(dir string) (io.Reader, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	// Create a new tar archive.
	tw := tar.NewWriter(buf)
	var err error
	if err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = path
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	}); err != nil {
		log.Print(err)
		return nil, err
	}

	return bytes.NewReader(buf.Bytes()), nil
}

func generateSingleFileTar(path string, input io.Reader) (io.Reader, error) {
	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	// TODO(loicalbertin) This is ugly! Read all the file to get its size
	var b []byte
	var err error
	if b, err = ioutil.ReadAll(input); err != nil {
		log.Print(err)
		return nil, err
	}

	hdr := &tar.Header{
		Name: filepath.Base(path),
		Mode: 0700,
		Size: int64(len(b)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		log.Print(err)
		return nil, err
	}
	if _, err := tw.Write([]byte(b)); err != nil {
		log.Print(err)
		return nil, err
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		log.Print(err)
		return nil, err
	}
	// Open the tar archive for reading.
	return bytes.NewReader(buf.Bytes()), nil
}
