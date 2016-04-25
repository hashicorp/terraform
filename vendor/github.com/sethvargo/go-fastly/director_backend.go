package fastly

import (
	"fmt"
	"time"
)

// DirectorBackend is the relationship between a director and a backend in the
// Fastly API.
type DirectorBackend struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Director  string     `mapstructure:"director_name"`
	Backend   string     `mapstructure:"backend_name"`
	CreatedAt *time.Time `mapstructure:"created_at"`
	UpdatedAt *time.Time `mapstructure:"updated_at"`
	DeletedAt *time.Time `mapstructure:"deleted_at"`
}

// CreateDirectorBackendInput is used as input to the CreateDirectorBackend
// function.
type CreateDirectorBackendInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Director is the name of the director (required).
	Director string

	// Backend is the name of the backend (required).
	Backend string
}

// CreateDirectorBackend creates a new Fastly backend.
func (c *Client) CreateDirectorBackend(i *CreateDirectorBackendInput) (*DirectorBackend, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Director == "" {
		return nil, ErrMissingDirector
	}

	if i.Backend == "" {
		return nil, ErrMissingBackend
	}

	path := fmt.Sprintf("/service/%s/version/%s/director/%s/backend/%s",
		i.Service, i.Version, i.Director, i.Backend)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var b *DirectorBackend
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// GetDirectorBackendInput is used as input to the GetDirectorBackend function.
type GetDirectorBackendInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Director is the name of the director (required).
	Director string

	// Backend is the name of the backend (required).
	Backend string
}

// GetDirectorBackend gets the backend configuration with the given parameters.
func (c *Client) GetDirectorBackend(i *GetDirectorBackendInput) (*DirectorBackend, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Director == "" {
		return nil, ErrMissingDirector
	}

	if i.Backend == "" {
		return nil, ErrMissingBackend
	}

	path := fmt.Sprintf("/service/%s/version/%s/director/%s/backend/%s",
		i.Service, i.Version, i.Director, i.Backend)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var b *DirectorBackend
	if err := decodeJSON(&b, resp.Body); err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteDirectorBackendInput is the input parameter to DeleteDirectorBackend.
type DeleteDirectorBackendInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Director is the name of the director (required).
	Director string

	// Backend is the name of the backend (required).
	Backend string
}

// DeleteDirectorBackend deletes the given backend version.
func (c *Client) DeleteDirectorBackend(i *DeleteDirectorBackendInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Director == "" {
		return ErrMissingDirector
	}

	if i.Backend == "" {
		return ErrMissingBackend
	}

	path := fmt.Sprintf("/service/%s/version/%s/director/%s/backend/%s",
		i.Service, i.Version, i.Director, i.Backend)
	resp, err := c.Delete(path, nil)
	if err != nil {
		return err
	}

	var r *statusResp
	if err := decodeJSON(&r, resp.Body); err != nil {
		return err
	}
	if !r.Ok() {
		return fmt.Errorf("Not Ok")
	}
	return nil
}
