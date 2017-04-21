// Copyright (c) 2012 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
)

// connWriter is used to write to a stream-oriented network connection.
type connWriter struct {
	innerWriter    io.WriteCloser
	reconnectOnMsg bool
	reconnect      bool
	net            string
	addr           string
	useTLS         bool
	configTLS      *tls.Config
}

// Creates writer to the address addr on the network netName.
// Connection will be opened on each write if reconnectOnMsg = true
func NewConnWriter(netName string, addr string, reconnectOnMsg bool) *connWriter {
	newWriter := new(connWriter)

	newWriter.net = netName
	newWriter.addr = addr
	newWriter.reconnectOnMsg = reconnectOnMsg

	return newWriter
}

// Creates a writer that uses SSL/TLS
func newTLSWriter(netName string, addr string, reconnectOnMsg bool, config *tls.Config) *connWriter {
	newWriter := new(connWriter)

	newWriter.net = netName
	newWriter.addr = addr
	newWriter.reconnectOnMsg = reconnectOnMsg
	newWriter.useTLS = true
	newWriter.configTLS = config

	return newWriter
}

func (connWriter *connWriter) Close() error {
	if connWriter.innerWriter == nil {
		return nil
	}

	return connWriter.innerWriter.Close()
}

func (connWriter *connWriter) Write(bytes []byte) (n int, err error) {
	if connWriter.neededConnectOnMsg() {
		err = connWriter.connect()
		if err != nil {
			return 0, err
		}
	}

	if connWriter.reconnectOnMsg {
		defer connWriter.innerWriter.Close()
	}

	n, err = connWriter.innerWriter.Write(bytes)
	if err != nil {
		connWriter.reconnect = true
	}

	return
}

func (connWriter *connWriter) String() string {
	return fmt.Sprintf("Conn writer: [%s, %s, %v]", connWriter.net, connWriter.addr, connWriter.reconnectOnMsg)
}

func (connWriter *connWriter) connect() error {
	if connWriter.innerWriter != nil {
		connWriter.innerWriter.Close()
		connWriter.innerWriter = nil
	}

	if connWriter.useTLS {
		conn, err := tls.Dial(connWriter.net, connWriter.addr, connWriter.configTLS)
		if err != nil {
			return err
		}
		connWriter.innerWriter = conn

		return nil
	}

	conn, err := net.Dial(connWriter.net, connWriter.addr)
	if err != nil {
		return err
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if ok {
		tcpConn.SetKeepAlive(true)
	}

	connWriter.innerWriter = conn

	return nil
}

func (connWriter *connWriter) neededConnectOnMsg() bool {
	if connWriter.reconnect {
		connWriter.reconnect = false
		return true
	}

	if connWriter.innerWriter == nil {
		return true
	}

	return connWriter.reconnectOnMsg
}
