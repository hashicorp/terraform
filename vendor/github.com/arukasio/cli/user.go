package arukas

import (
	// "errors"
	// "fmt"
	// "github.com/codegangsta/cli"
	// "os"
	"time"
)

// User represents a user data in struct variables.
type User struct {
	ID          string    `json:"-"`         // user id
	Name        string    `json:"name"`      // user name
	Email       string    `json:"email"`     // user e-mail
	Provider    string    `json:"provider"`  // user oAuth provider
	ImageURL    string    `json:"image_url"` // user profile image
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ConfirmedAt time.Time `json:"-"`
}

// GetID returns a stringified of an ID.
func (u User) GetID() string {
	return string(u.ID)
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface.
func (u *User) SetID(ID string) error {
	u.ID = ID
	return nil
}
