package ad

import (
	"fmt"
	"gopkg.in/ldap.v2"
	"log"
)

type Config struct {
	Domain   string
	IP       string
	Username string
	Password string
}

// Client() returns a connection for accessing AD services.
func (c *Config) Client() (*ldap.Conn, error) {
	var username string
	username = c.Username + "@" + c.Domain
	ad_conn, err := clientConnect(c.IP, username, c.Password)

	if err != nil {
		return nil, fmt.Errorf("Error while connection. Check IP, username or password: %s", err)
	}

	log.Printf("[INFO] AD connection successful for user: %s", c.Username)
	return ad_conn, nil
}

func clientConnect(ip, username, password string) (*ldap.Conn, error) {
	ad_conn, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ip, 389))
	if err != nil {
		return nil, err
	}

	err = ad_conn.Bind(username, password)
	if err != nil {
		return nil, err
	}
	return ad_conn, nil
}
