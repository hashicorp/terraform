package cmd

import (
	"bytes"
	"crypto/md5"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

type CmdClient struct {
	baseCmd            string
	statesTransferFile string
	lockTransferFile   string
	lockID             string
	jsonLockInfo       []byte
}

func (c *CmdClient) execCommand(arg string) error {
	args := []string{arg}
	cmd := exec.Command(c.baseCmd, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	log.Printf("[TRACE] backend/remote-state/cmd execCommand: %s\n%s", arg, out.String())
	return err
}

func logStart(action string) {
	log.Printf("[TRACE] backend/remote-state/cmd: starting %s operation", action)
}

func logResult(action string, err *error) {
	if *err == nil {
		log.Printf("[TRACE] backend/remote-state/cmd: exiting %s operation with success", action)
	} else {
		log.Printf("[TRACE] backend/remote-state/cmd: exiting %s operation with failure", action)
	}
}

func (c *CmdClient) Get() (*remote.Payload, error) {
	var err error
	logStart("Get")
	defer logResult("Get", &err)
	if err = c.execCommand("GET"); err != nil {
		return nil, err
	}

	file, err := os.Open(c.statesTransferFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()
	output, err := ioutil.ReadAll(file)

	hash := md5.Sum(output)
	payload := &remote.Payload{
		Data: output,
		MD5:  hash[:md5.Size],
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *CmdClient) Put(data []byte) error {
	var err error
	logStart("Put")
	defer logResult("Put", &err)
	err = ioutil.WriteFile(c.statesTransferFile, data, 0644)
	if err != nil {
		return err
	}
	err = c.execCommand("PUT")
	return err
}

func (c *CmdClient) Delete() error {
	var err error
	logStart("Delete")
	defer logResult("Delete", &err)
	err = c.execCommand("DELETE")
	return err
}

func (c *CmdClient) Unlock(id string) error {
	var err error
	logStart("Unlock")
	defer logResult("Unlock", &err)
	if c.lockTransferFile == "" {
		return nil
	}
	err = c.execCommand("UNLOCK")
	return err
}

func (c *CmdClient) Lock(info *state.LockInfo) (string, error) {
	var err error
	logStart("Lock")
	defer logResult("Lock", &err)
	if c.lockTransferFile == "" {
		return "", nil
	}
	c.lockID = ""

	jsonLockInfo := info.Marshal()
	err = ioutil.WriteFile(c.lockTransferFile, jsonLockInfo, 0644)
	if err != nil {
		return "", err
	}

	err = c.execCommand("LOCK")
	if err != nil {
		return "", err
	} else {
		c.lockID = info.ID
		c.jsonLockInfo = jsonLockInfo
		return info.ID, nil
	}
}
