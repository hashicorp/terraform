package fingerprint

import (
	"log"

	client "github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

// NomadFingerprint is used to fingerprint the Nomad version
type NomadFingerprint struct {
	StaticFingerprinter
	logger *log.Logger
}

// NewNomadFingerprint is used to create a Nomad fingerprint
func NewNomadFingerprint(logger *log.Logger) Fingerprint {
	f := &NomadFingerprint{logger: logger}
	return f
}

func (f *NomadFingerprint) Fingerprint(config *client.Config, node *structs.Node) (bool, error) {
	node.Attributes["nomad.version"] = config.Version
	node.Attributes["nomad.revision"] = config.Revision
	return true, nil
}
