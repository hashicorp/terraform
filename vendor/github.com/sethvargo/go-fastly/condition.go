package fastly

import (
	"fmt"
	"sort"
)

// Condition represents a condition response from the Fastly API.
type Condition struct {
	ServiceID string `mapstructure:"service_id"`
	Version   string `mapstructure:"version"`

	Name      string `mapstructure:"name"`
	Statement string `mapstructure:"statement"`
	Type      string `mapstructure:"type"`
	Priority  int    `mapstructure:"priority"`
}

// conditionsByName is a sortable list of conditions.
type conditionsByName []*Condition

// Len, Swap, and Less implement the sortable interface.
func (s conditionsByName) Len() int      { return len(s) }
func (s conditionsByName) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s conditionsByName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

// ListConditionsInput is used as input to the ListConditions function.
type ListConditionsInput struct {
	// Service is the ID of the service (required).
	Service string

	// Version is the specific configuration version (required).
	Version string
}

// ListConditions returns the list of conditions for the configuration version.
func (c *Client) ListConditions(i *ListConditionsInput) ([]*Condition, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/condition", i.Service, i.Version)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var cs []*Condition
	if err := decodeJSON(&cs, resp.Body); err != nil {
		return nil, err
	}
	sort.Stable(conditionsByName(cs))
	return cs, nil
}

// CreateConditionInput is used as input to the CreateCondition function.
type CreateConditionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	Name      string `form:"name,omitempty"`
	Statement string `form:"statement,omitempty"`
	Type      string `form:"type,omitempty"`
	Priority  int    `form:"priority,omitempty"`
}

// CreateCondition creates a new Fastly condition.
func (c *Client) CreateCondition(i *CreateConditionInput) (*Condition, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	path := fmt.Sprintf("/service/%s/version/%s/condition", i.Service, i.Version)
	resp, err := c.PostForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var co *Condition
	if err := decodeJSON(&co, resp.Body); err != nil {
		return nil, err
	}
	return co, nil
}

// GetConditionInput is used as input to the GetCondition function.
type GetConditionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the condition to fetch.
	Name string
}

// GetCondition gets the condition configuration with the given parameters.
func (c *Client) GetCondition(i *GetConditionInput) (*Condition, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/condition/%s", i.Service, i.Version, i.Name)
	resp, err := c.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var co *Condition
	if err := decodeJSON(&co, resp.Body); err != nil {
		return nil, err
	}
	return co, nil
}

// UpdateConditionInput is used as input to the UpdateCondition function.
type UpdateConditionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the condition to update.
	Name string

	Statement string `form:"statement,omitempty"`
	Type      string `form:"type,omitempty"`
	Priority  int    `form:"priority,omitempty"`
}

// UpdateCondition updates a specific condition.
func (c *Client) UpdateCondition(i *UpdateConditionInput) (*Condition, error) {
	if i.Service == "" {
		return nil, ErrMissingService
	}

	if i.Version == "" {
		return nil, ErrMissingVersion
	}

	if i.Name == "" {
		return nil, ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/condition/%s", i.Service, i.Version, i.Name)
	resp, err := c.PutForm(path, i, nil)
	if err != nil {
		return nil, err
	}

	var co *Condition
	if err := decodeJSON(&co, resp.Body); err != nil {
		return nil, err
	}
	return co, nil
}

// DeleteConditionInput is the input parameter to DeleteCondition.
type DeleteConditionInput struct {
	// Service is the ID of the service. Version is the specific configuration
	// version. Both fields are required.
	Service string
	Version string

	// Name is the name of the condition to delete (required).
	Name string
}

// DeleteCondition deletes the given condition version.
func (c *Client) DeleteCondition(i *DeleteConditionInput) error {
	if i.Service == "" {
		return ErrMissingService
	}

	if i.Version == "" {
		return ErrMissingVersion
	}

	if i.Name == "" {
		return ErrMissingName
	}

	path := fmt.Sprintf("/service/%s/version/%s/condition/%s", i.Service, i.Version, i.Name)
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
