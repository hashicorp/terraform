package rundeck

import (
	"encoding/xml"
)

// ProjectSummary provides the basic identifying information for a project within Rundeck.
type ProjectSummary struct {
	Name        string `xml:"name"`
	Description string `xml:"description,omitempty"`
	URL         string `xml:"url,attr"`
}

// Project represents a project within Rundeck.
type Project struct {
	Name        string `xml:"name"`
	Description string `xml:"description,omitempty"`

	// Config is the project configuration.
	//
	// When making requests, Config and RawConfigItems are combined to produce
	// a single set of configuration settings. Thus it isn't necessary and
	// doesn't make sense to duplicate the same properties in both properties.
	Config ProjectConfig `xml:"config"`

	// URL is used only to represent server responses. It is ignored when
	// making requests.
	URL string `xml:"url,attr"`

	// XMLName is used only in XML unmarshalling and doesn't need to
	// be set when creating a Project to send to the server.
	XMLName xml.Name `xml:"project"`
}

// ProjectConfig is a specialized map[string]string representing Rundeck project configuration
type ProjectConfig map[string]string

type projects struct {
	XMLName  xml.Name         `xml:"projects"`
	Count    int64            `xml:"count,attr"`
	Projects []ProjectSummary `xml:"project"`
}

// GetAllProjects retrieves and returns all of the projects defined in the Rundeck server.
func (c *Client) GetAllProjects() ([]ProjectSummary, error) {
	p := &projects{}
	err := c.get([]string{"projects"}, nil, p)
	return p.Projects, err
}

// GetProject retrieves and returns the named project.
func (c *Client) GetProject(name string) (*Project, error) {
	p := &Project{}
	err := c.get([]string{"project", name}, nil, p)
	return p, err
}

// CreateProject creates a new, empty project.
func (c *Client) CreateProject(project *Project) (*Project, error) {
	p := &Project{}
	err := c.post([]string{"projects"}, nil, project, p)
	return p, err
}

// DeleteProject deletes a project and all of its jobs.
func (c *Client) DeleteProject(name string) error {
	return c.delete([]string{"project", name})
}

// SetProjectConfig replaces the configuration of the named project.
func (c *Client) SetProjectConfig(projectName string, config ProjectConfig) error {
	return c.put(
		[]string{"project", projectName, "config"},
		config,
		nil,
	)
}

func (c ProjectConfig) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	rc := map[string]string(c)
	return marshalMapToXML(&rc, e, start, "property", "key", "value")
}

func (c *ProjectConfig) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	rc := (*map[string]string)(c)
	return unmarshalMapFromXML(rc, d, start, "property", "key", "value")
}
