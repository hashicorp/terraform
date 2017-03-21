package arukas

import (
	"errors"
	"time"
)

// App represents a application data in struct variables.
type App struct {
	ID          string     `json:"-"`
	Name        string     `json:"name"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"-"`
	ContainerID string     `json:"-"`
	Container   *Container `json:"-"`
	User        *User      `json:"-"`
}

// GetID returns a stringified of an ID.
func (a App) GetID() string {
	return string(a.ID)
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface.
func (a *App) SetID(ID string) error {
	a.ID = ID
	return nil
}

// SetToOneReferenceID sets the reference ID and satisfies the jsonapi.UnmarshalToOneRelations interface
func (a *App) SetToOneReferenceID(name, ID string) error {
	if name == "container" {
		if ID == "" {
			a.Container = nil
		} else {
			a.Container = &Container{ID: ID}
		}

		return nil
	} else if name == "user" {
		if ID == "" {
			a.User = nil
		} else {
			a.User = &User{ID: ID}
		}

		return nil
	}

	return errors.New("There is no to-one relationship with the name " + name)
}
