package mssql

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/denisenkom/go-mssqldb" //MS SQL db
	"github.com/hashicorp/go-multierror"
)

// Config - provider config
type Config struct {
	Host                   string
	Port                   int
	Encrypt                string
	TrustServerCertificate bool
	Certificate            string
	Username               string
	Password               string
}

// Client struct holding connection string
type Client struct {
	username string
	connStr  string
}

//NewClient returns new client config
func (c *Config) NewClient() (*Client, error) {
	// Connection String
	var connStr string
	// We need to validate some parameters
	var errs *multierror.Error

	err := c.ValidateEncrypt()
	if err != nil {
		errs = multierror.Append(errs, err)
	}

	connStr = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d", c.Host, c.Username, c.Password, c.Port)

	switch c.Encrypt {
	case "true":
		connStr += fmt.Sprintf(";encrypt=%s;TrustServerCertificate=%t", c.Encrypt, c.TrustServerCertificate)
		// If we have to check certificate we need certificate file
		if !c.TrustServerCertificate {
			if c.Certificate == "" {
				return nil, fmt.Errorf("Please provide full path to file that contains public key certificate of the CA that signed the SQL. You specified certificate = '%s'", c.Certificate)
			}
			if _, err = os.Stat(c.Certificate); os.IsNotExist(err) {
				return nil, fmt.Errorf("%s MS SQL certificate file cannot be found", c.Certificate)
			}
			connStr += fmt.Sprintf(";certificate=%s", c.Certificate)
		}
	case "disable":
		connStr += fmt.Sprintf(";encrypt=%s", c.Encrypt)
	}

	client := Client{
		connStr:  connStr,
		username: c.Username,
	}

	return &client, errs.ErrorOrNil()
}

//Connect will manually connect/diconnect to prevent a large number or db connections being made
func (c *Client) Connect() (*sql.DB, error) {
	db, err := sql.Open("mssql", c.connStr)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to MS SQL Server: %s", err)
	}

	return db, nil
}

// ValidateEncrypt returns error if the configured value for encrypt parameter is not a valid one
func (c *Config) ValidateEncrypt() error {
	var encryptValues = []string{"disable", "false", "true"}

	for _, valid := range encryptValues {
		if c.Encrypt == valid {
			return nil
		}
	}

	return fmt.Errorf("Allowed values for 'Encrypt' parameter are: disable, true, false. You specified %s", c.Encrypt)
}
