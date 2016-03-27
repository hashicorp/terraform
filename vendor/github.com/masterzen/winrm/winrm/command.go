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
	client    *Client
	shell     *Shell
	commandId string
	exitCode  int
	finished  bool
	err       error

	Stdin  *commandWriter
	Stdout *commandReader
	Stderr *commandReader

	done   chan struct{}
	cancel chan struct{}
}

func newCommand(shell *Shell, commandId string) *Command {
	command := &Command{shell: shell, client: shell.client, commandId: commandId, exitCode: 1, err: nil, done: make(chan struct{}), cancel: make(chan struct{})}
	command.Stdin = &commandWriter{Command: command, eof: false}
	command.Stdout = newCommandReader("stdout", command)
	command.Stderr = newCommandReader("stderr", command)

	go fetchOutput(command)

	return command
}

func newCommandReader(stream string, command *Command) *commandReader {
	read, write := io.Pipe()
	return &commandReader{Command: command, stream: stream, write: write, read: read}
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

func (command *Command) check() (err error) {
	if command.commandId == "" {
		return errors.New("Command has already been closed")
	}
	if command.shell == nil {
		return errors.New("Command has no associated shell")
	}
	if command.client == nil {
		return errors.New("Command has no associated client")
	}
	return
}

// Close will terminate the running command
func (command *Command) Close() (err error) {
	if err = command.check(); err != nil {
		return err
	}

	select { // close cancel channel if it's still open
	case <-command.cancel:
	default:
		close(command.cancel)
	}

	request := NewSignalRequest(command.client.url, command.shell.ShellId, command.commandId, &command.client.Parameters)
	defer request.Free()

	_, err = command.client.sendRequest(request)
	return err
}

func (command *Command) slurpAllOutput() (finished bool, err error) {
	if err = command.check(); err != nil {
		command.Stderr.write.CloseWithError(err)
		command.Stdout.write.CloseWithError(err)
		return true, err
	}

	request := NewGetOutputRequest(command.client.url, command.shell.ShellId, command.commandId, "stdout stderr", &command.client.Parameters)
	defer request.Free()

	response, err := command.client.sendRequest(request)
	if err != nil {
		if strings.Contains(err.Error(), "OperationTimeout") {
			// Operation timeout because there was no command output
			return
		}

		command.Stderr.write.CloseWithError(err)
		command.Stdout.write.CloseWithError(err)
		return true, err
	}

	var exitCode int
	var stdout, stderr bytes.Buffer
	finished, exitCode, err = ParseSlurpOutputErrResponse(response, &stdout, &stderr)
	if err != nil {
		command.Stderr.write.CloseWithError(err)
		command.Stdout.write.CloseWithError(err)
		return true, err
	}
	if stdout.Len() > 0 {
		command.Stdout.write.Write(stdout.Bytes())
	}
	if stderr.Len() > 0 {
		command.Stderr.write.Write(stderr.Bytes())
	}
	if finished {
		command.exitCode = exitCode
		command.Stderr.write.Close()
		command.Stdout.write.Close()
	}

	return
}

func (command *Command) sendInput(data []byte) (err error) {
	if err = command.check(); err != nil {
		return err
	}

	request := NewSendInputRequest(command.client.url, command.shell.ShellId, command.commandId, data, &command.client.Parameters)
	defer request.Free()

	_, err = command.client.sendRequest(request)
	return
}

// ExitCode returns command exit code when it is finished. Before that the result is always 0.
func (command *Command) ExitCode() int {
	return command.exitCode
}

// Calling this function will block the current goroutine until the remote command terminates.
func (command *Command) Wait() {
	// block until finished
	<-command.done
}

// Write data to this Pipe
func (w *commandWriter) Write(data []byte) (written int, err error) {
	for len(data) > 0 {
		if w.eof {
			err = io.EOF
			return
		}
		// never send more data than our EnvelopeSize.
		n := min(w.client.Parameters.EnvelopeSize-1000, len(data))
		if err = w.sendInput(data[:n]); err != nil {
			break
		}
		data = data[n:]
		written += int(n)
	}
	return
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

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
