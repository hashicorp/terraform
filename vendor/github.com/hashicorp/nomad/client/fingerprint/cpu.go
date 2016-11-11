package fingerprint

import (
	"fmt"
	"log"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/helper/stats"
	"github.com/hashicorp/nomad/nomad/structs"
)

// CPUFingerprint is used to fingerprint the CPU
type CPUFingerprint struct {
	StaticFingerprinter
	logger *log.Logger
}

// NewCPUFingerprint is used to create a CPU fingerprint
func NewCPUFingerprint(logger *log.Logger) Fingerprint {
	f := &CPUFingerprint{logger: logger}
	return f
}

func (f *CPUFingerprint) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {
	if err := stats.Init(); err != nil {
		return false, fmt.Errorf("Unable to obtain CPU information: %v", err)
	}

	modelName := stats.CPUModelName()
	if modelName != "" {
		node.Attributes["cpu.modelname"] = modelName
	}

	mhz := stats.CPUMHzPerCore()
	node.Attributes["cpu.frequency"] = fmt.Sprintf("%.0f", mhz)
	f.logger.Printf("[DEBUG] fingerprint.cpu: frequency: %.0f MHz", mhz)

	numCores := stats.CPUNumCores()
	node.Attributes["cpu.numcores"] = fmt.Sprintf("%d", numCores)
	f.logger.Printf("[DEBUG] fingerprint.cpu: core count: %d", numCores)

	tt := stats.TotalTicksAvailable()
	node.Attributes["cpu.totalcompute"] = fmt.Sprintf("%.0f", tt)

	if node.Resources == nil {
		node.Resources = &structs.Resources{}
	}

	node.Resources.CPU = int(tt)

	return true, nil
}
