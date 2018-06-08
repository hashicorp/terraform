package winrm

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

type commandWriter struct {
	*Command
	eof bool
}

type commandReader struct {
	*Command
	write  *io.PipeWriter
	read   *io.PipeReader
	stream string
}

// Command represents a given command running on a Shell. This structure allows to get access
// to the various stdout, stderr and stdin pipes.
type Command struct {
	client   *Client
	shell    *Shell
	id       string
	exitCode int
	finished bool
	err      error

	Stdin  *commandWriter
	Stdout *commandReader
	Stderr *commandReader

	done   chan struct{}
	cancel chan struct{}
}

func newCommand(shell *Shell, ids string) *Command {
	command := &Command{
		shell:    shell,
		client:   shell.client,
		id:       ids,
		exitCode: 0,
		err:      nil,
		done:     make(chan struct{}),
		cancel:   make(chan struct{}),
	}

	command.Stdout = newCommandReader("stdout", command)
	command.Stdin = &commandWriter{
		Command: command,
		eof:     false,
	}
	command.Stderr = newCommandReader("stderr", command)

	go fetchOutput(command)

	return command
}

func newCommandReader(stream string, command *Command) *commandReader {
	read, write := io.Pipe()
	return &commandReader{
		Command: command,
		stream:  stream,
		write:   write,
		read:    read,
	}
}

func fetchOutput(command *Command) {
	for {
		select {
		case <-command.cancel:
			close(command.done)
			return
		default:
			finished, err := command.slurpAllOutput()
			if finished {
				command.err = err
				close(command.done)
				return
			}
		}
	}
}

func (c *Command) check() error {
	if c.id == "" {
		return errors.New("Command has already been closed")
	}
	if c.shell == nil {
		return errors.New("Command has no associated shell")
	}
	if c.client == nil {
		return errors.New("Command has no associated client")
	}
	return nil
}

// Close will terminate the running command
func (c *Command) Close() error {
	if err := c.check(); err != nil {
		return err
	}

	select { // close cancel channel if it's still open
	case <-c.cancel:
	default:
		close(c.cancel)
	}

	request := NewSignalRequest(c.client.url, c.shell.id, c.id, &c.client.Parameters)
	defer request.Free()

	_, err := c.client.sendRequest(request)
	return err
}

func (c *Command) slurpAllOutput() (bool, error) {
	if err := c.check(); err != nil {
		c.Stderr.write.CloseWithError(err)
		c.Stdout.write.CloseWithError(err)
		return true, err
	}

	request := NewGetOutputRequest(c.client.url, c.shell.id, c.id, "stdout stderr", &c.client.Parameters)
	defer request.Free()

	response, err := c.client.sendRequest(request)
	if err != nil {
		if strings.Contains(err.Error(), "OperationTimeout") {
			// Operation timeout because there was no command output
			return false, err
		}
		if strings.Contains(err.Error(), "EOF") {
			c.exitCode = 16001
		}

		c.Stderr.write.CloseWithError(err)
		c.Stdout.write.CloseWithError(err)
		return true, err
	}

	var exitCode int
	var stdout, stderr bytes.Buffer
	finished, exitCode, err := ParseSlurpOutputErrResponse(response, &stdout, &stderr)
	if err != nil {
		c.Stderr.write.CloseWithError(err)
		c.Stdout.write.CloseWithError(err)
		return true, err
	}
	if stdout.Len() > 0 {
		c.Stdout.write.Write(stdout.Bytes())
	}
	if stderr.Len() > 0 {
		c.Stderr.write.Write(stderr.Bytes())
	}
	if finished {
		c.exitCode = exitCode
		c.Stderr.write.Close()
		c.Stdout.write.Close()
	}

	return finished, nil
}

func (c *Command) sendInput(data []byte) error {
	if err := c.check(); err != nil {
		return err
	}

	request := NewSendInputRequest(c.client.url, c.shell.id, c.id, data, &c.client.Parameters)
	defer request.Free()

	_, err := c.client.sendRequest(request)
	return err
}

// ExitCode returns command exit code when it is finished. Before that the result is always 0.
func (c *Command) ExitCode() int {
	return c.exitCode
}

// Wait function will block the current goroutine until the remote command terminates.
func (c *Command) Wait() {
	// block until finished
	<-c.done
}

// Write data to this Pipe
// commandWriter implements io.Writer interface
func (w *commandWriter) Write(data []byte) (int, error) {

	var (
		written int
		err     error
	)

	for len(data) > 0 {
		if w.eof {
			return written, io.EOF
		}
		// never send more data than our EnvelopeSize.
		n := min(w.client.Parameters.EnvelopeSize-1000, len(data))
		if err := w.sendInput(data[:n]); err != nil {
			break
		}
		data = data[n:]
		written += n
	}

	return written, err
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// Close method wrapper
// commandWriter implements io.Closer interface
func (w *commandWriter) Close() error {
	w.eof = true
	return w.Close()
}

// Read data from this Pipe
func (r *commandReader) Read(buf []byte) (int, error) {
	n, err := r.read.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	return n, err
}
