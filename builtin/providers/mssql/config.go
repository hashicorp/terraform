package mssql

import (
	"database/sql"
	"fmt"

	_ "github.com/denisenkom/go-mssqldb" //MS SQL db
)

// Config - provider config
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// Client struct holding connection string
type Client struct {
	username string
	connStr  string
}

//NewClient returns new client config
func (c *Config) NewClient() (*Client, error) {
	connStr := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d", c.Host, c.Username, c.Password, c.Port)

	client := Client{
		connStr:  connStr,
		username: c.Username,
	}

	return &client, nil
}

//Connect will manually connect/diconnect to prevent a large number or db connections being made
func (c *Client) Connect() (*sql.DB, error) {
	db, err := sql.Open("mssql", c.connStr)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to MS SQL Server: %s", err)
	}

	return db, nil
}
