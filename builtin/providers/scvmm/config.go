package scvmm

import (
	"log"

	"fmt"

	"strings"

	"github.com/masterzen/winrm"
)

// Config ... SCVMM configuration details
type Config struct {
	ServerIP string
	Port     int
	Username string
	Password string
}

//Connection ... Create a new connection with winrm to Powershell.
func (c *Config) Connection() (*winrm.Client, error) {

	endpoint := winrm.NewEndpoint(c.ServerIP, c.Port, false, false, nil, nil, nil, 0)
	winrmConnection, err := winrm.NewClient(endpoint, c.Username, c.Password)
	if err != nil {
		log.Printf("[ERROR] Failed to connect winrm: %v\n", err)
		return nil, err
	}

	shell, err := winrmConnection.CreateShell()
	if err != nil {
		log.Printf("[Error] While creating Shell %s", err)
		if strings.Contains(err.Error(), "http response error: 401") {
			return nil, fmt.Errorf("[Error] Please check whether username and password are correct.\n Error: %s", err.Error())
		} else if strings.Contains(err.Error(), "unknown error Post") {
			return nil, fmt.Errorf("[Error] Please check whether server ip and port number are correct.\n Error: %s", err.Error())
		} else {
			return nil, fmt.Errorf("[Error] While creating Shell %s", err)
		}
	}
	defer shell.Close()

	log.Printf("[INFO] Winrm connection successful")
	return winrmConnection, nil
}
