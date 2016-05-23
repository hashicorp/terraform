package docker

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"

	dc "github.com/fsouza/go-dockerclient"
)

const (
	// DefaultShebang is added at the top of a script file
	DefaultShebang = "#!/bin/sh\n"
)

// Communicator represents the SSH communicator
type Communicator struct {
	connInfo *connectionInfo
	client   *dc.Client
}

// New creates a new communicator implementation over SSH.
func New(s *terraform.InstanceState) (*Communicator, error) {
	connInfo, err := parseConnectionInfo(s)
	if err != nil {
		return nil, err
	}

	client, err := connInfo.NewClient()
	if err != nil {
		return nil, err
	}

	comm := &Communicator{
		connInfo: connInfo,
		client:   client,
	}

	return comm, nil
}

// Connect implementation of communicator.Communicator interface
func (c *Communicator) Connect(o terraform.UIOutput) (err error) {
	// No long-lived connection for Docker, since it's a request/response API
	log.Printf("[DEBUG] Connected docker communicator")
	return nil
}

// Disconnect implementation of communicator.Communicator interface
func (c *Communicator) Disconnect() error {
	log.Printf("[DEBUG] Disconnected docker communicator")
	return nil
}

// Timeout implementation of communicator.Communicator interface
func (c *Communicator) Timeout() time.Duration {
	// Our connection becomes available instantly, so this
	// doesn't actually mean anything and we just return a
	// fixed placeholder value.
	return time.Second * 10
}

// ScriptPath implementation of communicator.Communicator interface
func (c *Communicator) ScriptPath() string {
	return strings.Replace(
		c.connInfo.ScriptPath, "%RAND%",
		strconv.FormatInt(int64(rand.Int31()), 10), -1)
}

// Start implementation of communicator.Communicator interface
func (c *Communicator) Start(cmd *remote.Cmd) error {

	log.Printf("[DEBUG] docker exec: %s", cmd.Command)
	exec, err := c.client.CreateExec(dc.CreateExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Cmd:          []string{"/bin/sh", "-c", cmd.Command},
		Container:    c.connInfo.Container,
	})
	if err != nil {
		return err
	}

	// Start a goroutine to wait for the exec to end and set the
	// exit boolean and status.
	go func() {
		log.Printf("[DEBUG] docker communicator starting execution")
		err = c.client.StartExec(exec.ID, dc.StartExecOptions{
			InputStream:  cmd.Stdin,
			OutputStream: cmd.Stdout,
			ErrorStream:  cmd.Stderr,
		})
		if err != nil {
			return
		}

		log.Printf("[DEBUG] docker communicator execution ended")

		// Docker exec doesn't expose the exit status, so we will
		// just always return success. :/
		cmd.SetExited(0)
	}()

	return nil
}

func (c *Communicator) uploadFile(path string, mode int64, input io.Reader) error {
	log.Printf("[DEBUG] Uploading file to %s in docker container", path)
	targetDir := filepath.Dir(path)
	targetFile := filepath.Base(path)

	reader, writer := io.Pipe()

	// Start a goroutine to generate a tar stream that we'll write to
	// the docker server below.
	var writeErr error
	go func() {
		// Unfortunately we need to buffer the whole file in memory
		// to find out how big it is so we can write the tar header.
		var contents []byte
		log.Printf("[TRACE] Buffering content for %s", targetFile)
		contents, writeErr = ioutil.ReadAll(input)
		if writeErr != nil {
			return
		}

		tarWriter := tar.NewWriter(writer)
		defer tarWriter.Close()
		log.Printf("[TRACE] Writing tar header for %s", targetFile)
		writeErr = tarWriter.WriteHeader(&tar.Header{
			Name:     targetFile,
			Mode:     mode,
			Size:     int64(len(contents)),
			Typeflag: tar.TypeReg,
		})
		if writeErr != nil {
			return
		}

		log.Printf("[TRACE] Writing file contents for %s", targetFile)
		_, writeErr = tarWriter.Write(contents)
	}()

	log.Printf("[TRACE] Starting upload of %s", targetFile)
	err := c.client.UploadToContainer(c.connInfo.Container, dc.UploadToContainerOptions{
		InputStream: reader,
		Path:        targetDir,
	})
	if err != nil {
		return err
	}

	log.Printf("[TRACE] Upload of %s completed", targetFile)

	return writeErr
}

// Upload implementation of communicator.Communicator interface
func (c *Communicator) Upload(path string, input io.Reader) error {
	return c.uploadFile(path, 0655, input)
}

// UploadScript implementation of communicator.Communicator interface
func (c *Communicator) UploadScript(path string, input io.Reader) error {
	return c.uploadFile(path, 0755, input)
}

// UploadDir implementation of communicator.Communicator interface
func (c *Communicator) UploadDir(dst string, src string) error {
	log.Printf("[DEBUG] Uploading local dir %s to %s in docker container", src, dst)

	reader, writer := io.Pipe()

	// Start a goroutine to generate a tar stream that we'll write to
	// the docker server below.
	var writeErr error
	go func() {
		tarWriter := tar.NewWriter(writer)
		defer tarWriter.Close()

		writeErr = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Make the path relative to src, so in turn the docker server
			// will interpret it relative to the dst path.
			relPath, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}

			if info.IsDir() {
				return tarWriter.WriteHeader(&tar.Header{
					Name:     relPath,
					Mode:     int64(info.Mode()),
					Typeflag: tar.TypeDir,
				})
			}

			f, err := os.Open(path)
			defer f.Close()
			if err != nil {
				return err
			}

			stat, err := f.Stat()
			if err != nil {
				return err
			}

			err = tarWriter.WriteHeader(&tar.Header{
				Name:     relPath,
				Mode:     int64(info.Mode()),
				Size:     stat.Size(),
				Typeflag: tar.TypeReg,
			})
			if err != nil {
				return err
			}

			_, err = io.Copy(tarWriter, f)
			return err
		})
	}()

	err := c.client.UploadToContainer(c.connInfo.Container, dc.UploadToContainerOptions{
		InputStream: reader,
		Path:        dst,
	})
	if err != nil {
		return err
	}

	return writeErr
}
