package arukas

import (
	"errors"
	"fmt"
	"github.com/manyminds/api2go/jsonapi"
	"strconv"
	"strings"
	"time"
)

// PortMapping represents a docker container port mapping in struct variables.
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	ServicePort   int    `json:"service_port"`
	Host          string `json:"host"`
}

// TaskPorts is Multiple PortMapping.
type TaskPorts []PortMapping

// PortMappings is multiple TaskPorts.
type PortMappings []TaskPorts

// Env represents a docker container environment key-value in struct variables.
type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Envs is multiple Env.
type Envs []Env

// Port represents a docker protocol and port-number in struct variables.
type Port struct {
	Protocol string `json:"protocol"`
	Number   int    `json:"number"`
}

// Ports is multiple Port.
type Ports []Port

// Container represents a docker container data in struct variables.
type Container struct {
	Envs         Envs         `json:"envs"`
	Ports        Ports        `json:"ports"`
	PortMappings PortMappings `json:"port_mappings,omitempty"`
	StatusText   string       `json:"status_text,omitempty"`
	ID           string
	ImageName    string    `json:"image_name"`
	CreatedAt    JSONTime  `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	App          *App
	Mem          int    `json:"mem"`
	AppID        string `json:"app_id"`
	Instances    int    `json:"instances"`
	IsRunning    bool   `json:"is_running,omitempty"`
	Cmd          string `json:"cmd"`
	Name         string `json:"name"`
	Endpoint     string `json:"end_point,omitempty"`
}

// GetID returns a stringified of an ID.
func (c Container) GetID() string {
	return string(c.ID)
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface.
func (c *Container) SetID(ID string) error {
	c.ID = ID
	return nil
}

// GetReferences returns all related structs to transactions.
func (c Container) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "apps",
			Name: "app",
		},
	}
}

// GetReferencedIDs satisfies the jsonapi.MarshalLinkedRelations interface.
func (c Container) GetReferencedIDs() []jsonapi.ReferenceID {
	result := []jsonapi.ReferenceID{}

	if c.AppID != "" {
		result = append(result, jsonapi.ReferenceID{ID: c.AppID, Name: "app", Type: "apps"})
	}

	return result
}

// GetReferencedStructs to satisfy the jsonapi.MarhsalIncludedRelations interface.
func (c Container) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	result := []jsonapi.MarshalIdentifier{}
	if c.App != nil {
		result = append(result, c.App)
	}
	return result
}

// SetToOneReferenceID sets the reference ID and satisfies the jsonapi.UnmarshalToOneRelations interface.
func (c *Container) SetToOneReferenceID(name, ID string) error {
	if name == "app" {
		if ID == "" {
			c.App = nil
		} else {
			c.App = &App{ID: ID}
		}

		return nil
	}

	return errors.New("There is no to-one relationship with the name " + name)
}

// ParseEnv parse docker container envs.
func ParseEnv(envs []string) (Envs, error) {
	var parsedEnvs Envs
	for _, env := range envs {
		kv := strings.Split(env, "=")
		parsedEnvs = append(parsedEnvs, Env{Key: kv[0], Value: kv[1]})
	}
	return parsedEnvs, nil
}

// ParsePort parse docker container ports.
func ParsePort(ports []string) (Ports, error) {
	var parsedPorts Ports
	for _, port := range ports {
		kv := strings.Split(port, ":")
		num, err := strconv.Atoi(kv[0])
		if err != nil {
			return nil, fmt.Errorf("Port number must be numeric. Given: %s", kv[0])
		}
		if !(kv[1] == "tcp" || kv[1] == "udp") {
			return nil, fmt.Errorf("Port protocol must be \"tcp\" or \"udp\"")
		}
		parsedPorts = append(parsedPorts, Port{Number: num, Protocol: kv[1]})
	}
	return parsedPorts, nil
}
