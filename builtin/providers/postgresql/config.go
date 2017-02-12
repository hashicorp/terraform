package postgresql

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"unicode"

	"github.com/hashicorp/errwrap"
	_ "github.com/lib/pq" //PostgreSQL db
)

// Config - provider config
type Config struct {
	Host              string
	Port              int
	Database          string
	Username          string
	Password          string
	SSLMode           string
	ApplicationName   string
	Timeout           int
	ConnectTimeoutSec int
}

// Client struct holding connection string
type Client struct {
	username string
	connStr  string

	// PostgreSQL lock on pg_catalog.  Many of the operations that Terraform
	// performs are not permitted to be concurrent.  Unlike traditional
	// PostgreSQL tables that use MVCC, many of the PostgreSQL system
	// catalogs look like tables, but are not in-fact able to be
	// concurrently updated.
	catalogLock sync.RWMutex
}

// NewClient returns new client config
func (c *Config) NewClient() (*Client, error) {
	// NOTE: dbname must come before user otherwise dbname will be set to
	// user.
	const dsnFmt = "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s fallback_application_name=%s connect_timeout=%d"

	// Quote empty strings or strings that contain whitespace
	q := func(s string) string {
		b := bytes.NewBufferString(`'`)
		b.Grow(len(s) + 2)
		var haveWhitespace bool
		for _, r := range s {
			if unicode.IsSpace(r) {
				haveWhitespace = true
			}

			switch r {
			case '\'':
				b.WriteString(`\'`)
			case '\\':
				b.WriteString(`\\`)
			default:
				b.WriteRune(r)
			}
		}

		b.WriteString(`'`)

		str := b.String()
		if haveWhitespace || len(str) == 2 {
			return str
		}
		return str[1 : len(str)-1]
	}

	logDSN := fmt.Sprintf(dsnFmt, q(c.Host), c.Port, q(c.Database), q(c.Username), q("<redacted>"), q(c.SSLMode), q(c.ApplicationName), c.ConnectTimeoutSec)
	log.Printf("[INFO] PostgreSQL DSN: `%s`", logDSN)

	connStr := fmt.Sprintf(dsnFmt, q(c.Host), c.Port, q(c.Database), q(c.Username), q(c.Password), q(c.SSLMode), q(c.ApplicationName), c.ConnectTimeoutSec)
	client := Client{
		connStr:  connStr,
		username: c.Username,
	}

	return &client, nil
}

// Connect will manually connect/disconnect to prevent a large
// number or db connections being made
func (c *Client) Connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.connStr)
	if err != nil {
		return nil, errwrap.Wrapf("Error connecting to PostgreSQL server: {{err}}", err)
	}

	return db, nil
}
