// +build !windows

package logging

import (
	"bufio"
	"log"
	"net"
	"sync"
)

// SyslogServer is a server which listens to syslog messages and parses them
type SyslogServer struct {
	listener net.Listener
	messages chan *SyslogMessage
	parser   *DockerLogParser

	doneCh   chan interface{}
	done     bool
	doneLock sync.Mutex

	logger *log.Logger
}

// NewSyslogServer creates a new syslog server
func NewSyslogServer(l net.Listener, messages chan *SyslogMessage, logger *log.Logger) *SyslogServer {
	parser := NewDockerLogParser(logger)
	return &SyslogServer{
		listener: l,
		messages: messages,
		parser:   parser,
		logger:   logger,
		doneCh:   make(chan interface{}),
	}
}

// Start starts accepting syslog connections
func (s *SyslogServer) Start() {
	for {
		select {
		case <-s.doneCh:
			s.listener.Close()
			return
		default:
			connection, err := s.listener.Accept()
			if err != nil {
				s.logger.Printf("[ERR] logcollector.server: error in accepting connection: %v", err)
				continue
			}
			go s.read(connection)
		}
	}
}

// read reads the bytes from a connection
func (s *SyslogServer) read(connection net.Conn) {
	defer connection.Close()
	scanner := bufio.NewScanner(bufio.NewReader(connection))

	for {
		select {
		case <-s.doneCh:
			return
		default:
		}
		if scanner.Scan() {
			b := scanner.Bytes()
			msg := s.parser.Parse(b)
			s.messages <- msg
		} else {
			return
		}
	}
}

// Shutdown shutsdown the syslog server
func (s *SyslogServer) Shutdown() {
	s.doneLock.Lock()
	s.doneLock.Unlock()

	if !s.done {
		close(s.doneCh)
		close(s.messages)
		s.done = true
	}
}
