package google

import (
	"fmt"

	"google.golang.org/api/compute/v1"
)

const FINGERPRINT_RETRIES = 10
const FINGERPRINT_FAIL = "Invalid fingerprint."

// Since the google compute API uses optimistic locking, there is a chance
// we need to resubmit our updated metadata. To do this, you need to provide
// an update function that attempts to submit your metadata
func MetadataRetryWrapper(update func() error) error {
	attempt := 0
	for attempt < FINGERPRINT_RETRIES {
		err := update()
		if err != nil && err.Error() == FINGERPRINT_FAIL {
			attempt++
		} else {
			return err
		}
	}

	return fmt.Errorf("Failed to update metadata after %d retries", attempt)
}

// Update the metadata (serverMD) according to the provided diff (oldMDMap v
// newMDMap).
func MetadataUpdate(oldMDMap map[string]interface{}, newMDMap map[string]interface{}, serverMD *compute.Metadata) {
	curMDMap := make(map[string]string)
	// Load metadata on server into map
	for _, kv := range serverMD.Items {
		// If the server state has a key that we had in our old
		// state, but not in our new state, we should delete it
		_, okOld := oldMDMap[kv.Key]
		_, okNew := newMDMap[kv.Key]
		if okOld && !okNew {
			continue
		} else {
			curMDMap[kv.Key] = *kv.Value
		}
	}

	// Insert new metadata into existing metadata (overwriting when needed)
	for key, val := range newMDMap {
		curMDMap[key] = val.(string)
	}

	// Reformat old metadata into a list
	serverMD.Items = nil
	for key, val := range curMDMap {
		v := val
		serverMD.Items = append(serverMD.Items, &compute.MetadataItems{
			Key:   key,
			Value: &v,
		})
	}
}

// Format metadata from the server data format -> schema data format
func MetadataFormatSchema(curMDMap map[string]interface{}, md *compute.Metadata) map[string]interface{} {
	newMD := make(map[string]interface{})

	for _, kv := range md.Items {
		if _, ok := curMDMap[kv.Key]; ok {
			newMD[kv.Key] = *kv.Value
		}
	}

	return newMD
}
