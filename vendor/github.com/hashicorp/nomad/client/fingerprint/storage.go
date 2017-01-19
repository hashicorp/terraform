package fingerprint

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hashicorp/nomad/client/config"
	"github.com/hashicorp/nomad/nomad/structs"
)

const bytesPerMegabyte = 1024 * 1024

// StorageFingerprint is used to measure the amount of storage free for
// applications that the Nomad agent will run on this machine.
type StorageFingerprint struct {
	StaticFingerprinter
	logger *log.Logger
}

func NewStorageFingerprint(logger *log.Logger) Fingerprint {
	fp := &StorageFingerprint{logger: logger}
	return fp
}

func (f *StorageFingerprint) Fingerprint(cfg *config.Config, node *structs.Node) (bool, error) {

	// Initialize these to empty defaults
	node.Attributes["unique.storage.volume"] = ""
	node.Attributes["unique.storage.bytestotal"] = ""
	node.Attributes["unique.storage.bytesfree"] = ""
	if node.Resources == nil {
		node.Resources = &structs.Resources{}
	}

	// Guard against unset AllocDir
	storageDir := cfg.AllocDir
	if storageDir == "" {
		var err error
		storageDir, err = os.Getwd()
		if err != nil {
			return false, fmt.Errorf("unable to get CWD from filesystem: %s", err)
		}
	}

	volume, total, free, err := f.diskFree(storageDir)
	if err != nil {
		return false, fmt.Errorf("failed to determine disk space for %s: %v", storageDir, err)
	}

	node.Attributes["unique.storage.volume"] = volume
	node.Attributes["unique.storage.bytestotal"] = strconv.FormatUint(total, 10)
	node.Attributes["unique.storage.bytesfree"] = strconv.FormatUint(free, 10)

	node.Resources.DiskMB = int(free / bytesPerMegabyte)

	return true, nil
}
