// Copyright (C) 2015 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

// scw helpers

// Package utils contains helpers
package utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/Sirupsen/logrus"
	log "github.com/Sirupsen/logrus"
	"github.com/mattn/go-isatty"
	"github.com/moul/gotty-client"
	"github.com/scaleway/scaleway-cli/pkg/sshcommand"
)

// SpawnRedirection is used to redirects the fluxes
type SpawnRedirection struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// SSHExec executes a command over SSH and redirects file-descriptors
func SSHExec(publicIPAddress, privateIPAddress, user string, port int, command []string, checkConnection bool, gateway string, enableSSHKeyForwarding bool) error {
	gatewayUser := "root"
	gatewayIPAddress := gateway
	if strings.Contains(gateway, "@") {
		parts := strings.Split(gatewayIPAddress, "@")
		if len(parts) != 2 {
			return fmt.Errorf("gateway: must be like root@IP")
		}
		gatewayUser = parts[0]
		gatewayIPAddress = parts[1]
		gateway = gatewayUser + "@" + gatewayIPAddress
	}

	if publicIPAddress == "" && gatewayIPAddress == "" {
		return errors.New("server does not have public IP")
	}
	if privateIPAddress == "" && gatewayIPAddress != "" {
		return errors.New("server does not have private IP")
	}

	if checkConnection {
		useGateway := gatewayIPAddress != ""
		if useGateway && !IsTCPPortOpen(fmt.Sprintf("%s:22", gatewayIPAddress)) {
			return errors.New("gateway is not available, try again later")
		}
		if !useGateway && !IsTCPPortOpen(fmt.Sprintf("%s:%d", publicIPAddress, port)) {
			return errors.New("server is not ready, try again later")
		}
	}

	sshCommand := NewSSHExecCmd(publicIPAddress, privateIPAddress, user, port, isatty.IsTerminal(os.Stdin.Fd()), command, gateway, enableSSHKeyForwarding)

	log.Debugf("Executing: %s", sshCommand)

	spawn := exec.Command("ssh", sshCommand.Slice()[1:]...)
	spawn.Stdout = os.Stdout
	spawn.Stdin = os.Stdin
	spawn.Stderr = os.Stderr
	return spawn.Run()
}

// NewSSHExecCmd computes execve compatible arguments to run a command via ssh
func NewSSHExecCmd(publicIPAddress, privateIPAddress, user string, port int, allocateTTY bool, command []string, gatewayIPAddress string, enableSSHKeyForwarding bool) *sshcommand.Command {
	quiet := os.Getenv("DEBUG") != "1"
	secureExec := os.Getenv("SCW_SECURE_EXEC") == "1"
	sshCommand := &sshcommand.Command{
		AllocateTTY:         allocateTTY,
		Command:             command,
		Host:                publicIPAddress,
		Quiet:               quiet,
		SkipHostKeyChecking: !secureExec,
		User:                user,
		NoEscapeCommand:     true,
		Port:                port,
		EnableSSHKeyForwarding: enableSSHKeyForwarding,
	}
	if gatewayIPAddress != "" {
		sshCommand.Host = privateIPAddress
		sshCommand.Gateway = &sshcommand.Command{
			Host:                gatewayIPAddress,
			SkipHostKeyChecking: !secureExec,
			AllocateTTY:         allocateTTY,
			Quiet:               quiet,
			User:                user,
			Port:                port,
		}
	}

	return sshCommand
}

// GeneratingAnSSHKey generates an SSH key
func GeneratingAnSSHKey(cfg SpawnRedirection, path string, name string) (string, error) {
	args := []string{
		"-t",
		"rsa",
		"-b",
		"4096",
		"-f",
		filepath.Join(path, name),
		"-N",
		"",
		"-C",
		"",
	}
	log.Infof("Executing commands %v", args)
	spawn := exec.Command("ssh-keygen", args...)
	spawn.Stdout = cfg.Stdout
	spawn.Stdin = cfg.Stdin
	spawn.Stderr = cfg.Stderr
	return args[5], spawn.Run()
}

// WaitForTCPPortOpen calls IsTCPPortOpen in a loop
func WaitForTCPPortOpen(dest string) error {
	for {
		if IsTCPPortOpen(dest) {
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// IsTCPPortOpen returns true if a TCP communication with "host:port" can be initialized
func IsTCPPortOpen(dest string) bool {
	conn, err := net.DialTimeout("tcp", dest, time.Duration(2000)*time.Millisecond)
	if err == nil {
		defer conn.Close()
	}
	return err == nil
}

// TruncIf ensures the input string does not exceed max size if cond is met
func TruncIf(str string, max int, cond bool) string {
	if cond && len(str) > max {
		return str[:max]
	}
	return str
}

// Wordify convert complex name to a single word without special shell characters
func Wordify(str string) string {
	str = regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(str, "_")
	str = regexp.MustCompile(`__+`).ReplaceAllString(str, "_")
	str = strings.Trim(str, "_")
	return str
}

// PathToTARPathparts returns the two parts of a unix path
func PathToTARPathparts(fullPath string) (string, string) {
	fullPath = strings.TrimRight(fullPath, "/")
	return path.Dir(fullPath), path.Base(fullPath)
}

// RemoveDuplicates transforms an array into a unique array
func RemoveDuplicates(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key := range encountered {
		result = append(result, key)
	}
	return result
}

// AttachToSerial tries to connect to server serial using 'gotty-client' and fallback with a help message
func AttachToSerial(serverID, apiToken, url string) (*gottyclient.Client, chan bool, error) {
	gottyURL := os.Getenv("SCW_GOTTY_URL")
	if gottyURL == "" {
		gottyURL = url
	}
	URL := fmt.Sprintf("%s?arg=%s&arg=%s", gottyURL, apiToken, serverID)

	logrus.Debug("Connection to ", URL)
	gottycli, err := gottyclient.NewClient(URL)
	if err != nil {
		return nil, nil, err
	}

	if os.Getenv("SCW_TLSVERIFY") == "0" {
		gottycli.SkipTLSVerify = true
	}

	gottycli.UseProxyFromEnv = true

	if err = gottycli.Connect(); err != nil {
		return nil, nil, err
	}
	done := make(chan bool)

	fmt.Println("You are connected, type 'Ctrl+q' to quit.")
	go func() {
		gottycli.Loop()
		gottycli.Close()
		done <- true
	}()
	return gottycli, done, nil
}

func rfc4716hex(data []byte) string {
	fingerprint := ""

	for i := 0; i < len(data); i++ {
		fingerprint = fmt.Sprintf("%s%0.2x", fingerprint, data[i])
		if i != len(data)-1 {
			fingerprint = fingerprint + ":"
		}
	}
	return fingerprint
}

// SSHGetFingerprint returns the fingerprint of an SSH key
func SSHGetFingerprint(key []byte) (string, error) {
	publicKey, comment, _, _, err := ssh.ParseAuthorizedKey(key)
	if err != nil {
		return "", err
	}
	switch reflect.TypeOf(publicKey).String() {
	case "*ssh.rsaPublicKey", "*ssh.dsaPublicKey", "*ssh.ecdsaPublicKey":
		md5sum := md5.Sum(publicKey.Marshal())
		return publicKey.Type() + " " + rfc4716hex(md5sum[:]) + " " + comment, nil
	default:
		return "", errors.New("Can't handle this key")
	}
}
