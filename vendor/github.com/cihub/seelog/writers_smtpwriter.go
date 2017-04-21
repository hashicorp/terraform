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
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"path/filepath"
	"strings"
)

const (
	// Default subject phrase for sending emails.
	DefaultSubjectPhrase = "Diagnostic message from server: "

	// Message subject pattern composed according to RFC 5321.
	rfc5321SubjectPattern = "From: %s <%s>\nSubject: %s\n\n"
)

// smtpWriter is used to send emails via given SMTP-server.
type smtpWriter struct {
	auth               smtp.Auth
	hostName           string
	hostPort           string
	hostNameWithPort   string
	senderAddress      string
	senderName         string
	recipientAddresses []string
	caCertDirPaths     []string
	mailHeaders        []string
	subject            string
}

// NewSMTPWriter returns a new SMTP-writer.
func NewSMTPWriter(sa, sn string, ras []string, hn, hp, un, pwd string, cacdps []string, subj string, headers []string) *smtpWriter {
	return &smtpWriter{
		auth:               smtp.PlainAuth("", un, pwd, hn),
		hostName:           hn,
		hostPort:           hp,
		hostNameWithPort:   fmt.Sprintf("%s:%s", hn, hp),
		senderAddress:      sa,
		senderName:         sn,
		recipientAddresses: ras,
		caCertDirPaths:     cacdps,
		subject:            subj,
		mailHeaders:        headers,
	}
}

func prepareMessage(senderAddr, senderName, subject string, body []byte, headers []string) []byte {
	headerLines := fmt.Sprintf(rfc5321SubjectPattern, senderName, senderAddr, subject)
	// Build header lines if configured.
	if headers != nil && len(headers) > 0 {
		headerLines += strings.Join(headers, "\n")
		headerLines += "\n"
	}
	return append([]byte(headerLines), body...)
}

// getTLSConfig gets paths of PEM files with certificates,
// host server name and tries to create an appropriate TLS.Config.
func getTLSConfig(pemFileDirPaths []string, hostName string) (config *tls.Config, err error) {
	if pemFileDirPaths == nil || len(pemFileDirPaths) == 0 {
		err = errors.New("invalid PEM file paths")
		return
	}
	pemEncodedContent := []byte{}
	var (
		e     error
		bytes []byte
	)
	// Create a file-filter-by-extension, set aside non-pem files.
	pemFilePathFilter := func(fp string) bool {
		if filepath.Ext(fp) == ".pem" {
			return true
		}
		return false
	}
	for _, pemFileDirPath := range pemFileDirPaths {
		pemFilePaths, err := getDirFilePaths(pemFileDirPath, pemFilePathFilter, false)
		if err != nil {
			return nil, err
		}

		// Put together all the PEM files to decode them as a whole byte slice.
		for _, pfp := range pemFilePaths {
			if bytes, e = ioutil.ReadFile(pfp); e == nil {
				pemEncodedContent = append(pemEncodedContent, bytes...)
			} else {
				return nil, fmt.Errorf("cannot read file: %s: %s", pfp, e.Error())
			}
		}
	}
	config = &tls.Config{RootCAs: x509.NewCertPool(), ServerName: hostName}
	isAppended := config.RootCAs.AppendCertsFromPEM(pemEncodedContent)
	if !isAppended {
		// Extract this into a separate error.
		err = errors.New("invalid PEM content")
		return
	}
	return
}

// SendMail accepts TLS configuration, connects to the server at addr,
// switches to TLS if possible, authenticates with mechanism a if possible,
// and then sends an email from address from, to addresses to, with message msg.
func sendMailWithTLSConfig(config *tls.Config, addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	// Check if the server supports STARTTLS extension.
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	// Check if the server supports AUTH extension and use given smtp.Auth.
	if a != nil {
		if isSupported, _ := c.Extension("AUTH"); isSupported {
			if err = c.Auth(a); err != nil {
				return err
			}
		}
	}
	// Portion of code from the official smtp.SendMail function,
	// see http://golang.org/src/pkg/net/smtp/smtp.go.
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}

// Write pushes a text message properly composed according to RFC 5321
// to a post server, which sends it to the recipients.
func (smtpw *smtpWriter) Write(data []byte) (int, error) {
	var err error

	if smtpw.caCertDirPaths == nil {
		err = smtp.SendMail(
			smtpw.hostNameWithPort,
			smtpw.auth,
			smtpw.senderAddress,
			smtpw.recipientAddresses,
			prepareMessage(smtpw.senderAddress, smtpw.senderName, smtpw.subject, data, smtpw.mailHeaders),
		)
	} else {
		config, e := getTLSConfig(smtpw.caCertDirPaths, smtpw.hostName)
		if e != nil {
			return 0, e
		}
		err = sendMailWithTLSConfig(
			config,
			smtpw.hostNameWithPort,
			smtpw.auth,
			smtpw.senderAddress,
			smtpw.recipientAddresses,
			prepareMessage(smtpw.senderAddress, smtpw.senderName, smtpw.subject, data, smtpw.mailHeaders),
		)
	}
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

// Close closes down SMTP-connection.
func (smtpw *smtpWriter) Close() error {
	// Do nothing as Write method opens and closes connection automatically.
	return nil
}
