// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

type pluginSHA256LockFile struct {
	Filename string
}

// Read loads the lock information from the file and returns it. If the file
// cannot be read, an empty map is returned to indicate that _no_ providers
// are acceptable, since the user must run "terraform init" to lock some
// providers before a context can be created.
func (pf *pluginSHA256LockFile) Read() map[string][]byte {
	// Returning an empty map is different than nil because it causes
	// us to reject all plugins as uninitialized, rather than applying no
	// constraints at all.
	//
	// We don't surface any specific errors here because we want it to all
	// roll up into our more-user-friendly error that appears when plugin
	// constraint verification fails during context creation.
	digests := make(map[string][]byte)

	buf, err := ioutil.ReadFile(pf.Filename)
	if err != nil {
		// This is expected if the user runs any context-using command before
		// running "terraform init".
		log.Printf("[INFO] Failed to read plugin lock file %s: %s", pf.Filename, err)
		return digests
	}

	var strDigests map[string]string
	err = json.Unmarshal(buf, &strDigests)
	if err != nil {
		// This should never happen unless the user directly edits the file.
		log.Printf("[WARN] Plugin lock file %s failed to parse as JSON: %s", pf.Filename, err)
		return digests
	}

	for name, strDigest := range strDigests {
		var digest []byte
		_, err := fmt.Sscanf(strDigest, "%x", &digest)
		if err == nil {
			digests[name] = digest
		} else {
			// This should never happen unless the user directly edits the file.
			log.Printf("[WARN] Plugin lock file %s has invalid digest for %q", pf.Filename, name)
		}
	}

	return digests
}
