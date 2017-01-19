package fingerprint

import (
	"fmt"
	"log"
	"strconv"
	"time"

	consul "github.com/hashicorp/consul/api"

	client "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

const (
	consulAvailable   = "available"
	consulUnavailable = "unavailable"
)

// ConsulFingerprint is used to fingerprint the architecture
type ConsulFingerprint struct {
	logger    *log.Logger
	client    *consul.Client
	lastState string
}

// NewConsulFingerprint is used to create an OS fingerprint
func NewConsulFingerprint(logger *log.Logger) Fingerprint {
	return &ConsulFingerprint{logger: logger, lastState: consulUnavailable}
}

func (f *ConsulFingerprint) Fingerprint(config *client.Config, node *structs.Node) (bool, error) {
	// Guard against uninitialized Links
	if node.Links == nil {
		node.Links = map[string]string{}
	}

	// Only create the client once to avoid creating too many connections to
	// Consul.
	if f.client == nil {
		consulConfig, err := config.ConsulConfig.ApiConfig()
		if err != nil {
			return false, fmt.Errorf("Failed to initialize the Consul client config: %v", err)
		}

		f.client, err = consul.NewClient(consulConfig)
		if err != nil {
			return false, fmt.Errorf("Failed to initialize consul client: %s", err)
		}
	}

	// We'll try to detect consul by making a query to to the agent's self API.
	// If we can't hit this URL consul is probably not running on this machine.
	info, err := f.client.Agent().Self()
	if err != nil {
		// Clear any attributes set by a previous fingerprint.
		f.clearConsulAttributes(node)

		// Print a message indicating that the Consul Agent is not available
		// anymore
		if f.lastState == consulAvailable {
			f.logger.Printf("[INFO] fingerprint.consul: consul agent is unavailable")
		}
		f.lastState = consulUnavailable
		return false, nil
	}

	node.Attributes["consul.server"] = strconv.FormatBool(info["Config"]["Server"].(bool))
	node.Attributes["consul.version"] = info["Config"]["Version"].(string)
	node.Attributes["consul.revision"] = info["Config"]["Revision"].(string)
	node.Attributes["unique.consul.name"] = info["Config"]["NodeName"].(string)
	node.Attributes["consul.datacenter"] = info["Config"]["Datacenter"].(string)

	node.Links["consul"] = fmt.Sprintf("%s.%s",
		node.Attributes["consul.datacenter"],
		node.Attributes["unique.consul.name"])

	// If the Consul Agent was previously unavailable print a message to
	// indicate the Agent is available now
	if f.lastState == consulUnavailable {
		f.logger.Printf("[INFO] fingerprint.consul: consul agent is available")
	}
	f.lastState = consulAvailable
	return true, nil
}

// clearConsulAttributes removes consul attributes and links from the passed
// Node.
func (f *ConsulFingerprint) clearConsulAttributes(n *structs.Node) {
	delete(n.Attributes, "consul.server")
	delete(n.Attributes, "consul.version")
	delete(n.Attributes, "consul.revision")
	delete(n.Attributes, "unique.consul.name")
	delete(n.Attributes, "consul.datacenter")
	delete(n.Links, "consul")
}

func (f *ConsulFingerprint) Periodic() (bool, time.Duration) {
	return true, 15 * time.Second
}
