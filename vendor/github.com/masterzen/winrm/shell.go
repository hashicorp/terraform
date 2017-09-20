package winrm

// Shell is the local view of a WinRM Shell of a given Client
type Shell struct {
	client *Client
	id     string
}

// Execute command on the given Shell, returning either an error or a Command
func (s *Shell) Execute(command string, arguments ...string) (*Command, error) {
	request := NewExecuteCommandRequest(s.client.url, s.id, command, arguments, &s.client.Parameters)
	defer request.Free()

	response, err := s.client.sendRequest(request)
	if err != nil {
		return nil, err
	}

	commandID, err := ParseExecuteCommandResponse(response)
	if err != nil {
		return nil, err
	}

	cmd := newCommand(s, commandID)

	return cmd, nil
}

// Close will terminate this shell. No commands can be issued once the shell is closed.
func (s *Shell) Close() error {
	request := NewDeleteShellRequest(s.client.url, s.id, &s.client.Parameters)
	defer request.Free()

	_, err := s.client.sendRequest(request)
	return err
}
