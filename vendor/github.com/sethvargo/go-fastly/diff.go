package fastly

import "fmt"

// Diff represents a diff of two versions as a response from the Fastly API.
type Diff struct {
	Format string `mapstructure:"format"`
	From   string `mapstructure:"from"`
	To     string `mapstructure:"to"`
	Diff   string `mapstructure:"diff"`
}

// GetDiffInput is used as input to the GetDiff function.
type GetDiffInput struct {
	// Service is the ID of the service (required).
	Service string

	// From is the version to diff from. This can either be a string indicating a
	// positive number (e.g. "1") or a negative number from "-1" down ("-1" is the
	// latest version).
	From string

	// To is the version to diff up to. The same rules for From apply.
	To string

	// Format is an optional field to specify the format with which the diff will
	// be returned. Acceptable values are "text" (default), "html", or
	// "html_simple".
	Format string
}

// GetDiff returns the diff of the given versions.
func (c *Client) GetDiff(i *GetDiffInput) (*Diff, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.From == "" {
		return nil, ErrMissingFrom
	}

	if i.To == "" {
		return nil, ErrMissingTo
	}

	path := fmt.Sprintf("service/%s/diff/from/%s/to/%s", i.Service, i.From, i.To)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var d *Diff
	if err := decodeJSON(&d, resp.Body); err != nil {
		return nil, err
	}
	return d, nil
}
