package postgresql

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
}

// NewClient() return new db conn
func (c *Config) NewClient() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s", c.Host, c.Port, c.Username, c.Password)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to postgresql server: %s", err)
	}

	return db, nil
}
