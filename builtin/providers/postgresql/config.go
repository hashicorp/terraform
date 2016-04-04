package postgresql

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" //PostgreSQL db
)

// Config - provider config
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	SslMode  string
}

// Client struct holding connection string
type Client struct {
	username string
	connStr  string
}

//NewClient returns new client config
func (c *Config) NewClient() (*Client, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s", c.Host, c.Port, c.Username, c.Password, c.SslMode)

	client := Client{
		connStr:  connStr,
		username: c.Username,
	}

	return &client, nil
}

//Connect will manually connect/diconnect to prevent a large number or db connections being made
func (c *Client) Connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.connStr)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to postgresql server: %s", err)
	}

	return db, nil
}
