package gcs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"golang.org/x/net/context"
)

// remoteClient is used by "state/remote".State to read and write
// blobs representing state.
// Implements "state/remote".ClientLocker
type remoteClient struct {
	mutex sync.Mutex

	storageContext context.Context
	storageClient  *storage.Client
	bucketName     string
	stateFilePath  string
	lockFilePath   string
	encryptionKey  []byte

	// The initial generation number of the lock file created by this
	// remoteClient.
	generation *int64

	// Channel used for signalling the lock-heartbeating goroutine to stop.
	stopHeartbeatCh chan bool
}

// Name of the metadata header on the lock file which indicates that the lock
// file has been created by a client which is supposed to periodically perform
// heartbeats on it. This header facilitates a safe migration from previous
// Terraform versions that do not yet perform any heartbeats on the lock file.
// A lock file will only be considered stale and force-unlocked if it's age
// exceeds minHeartbeatAgeUntilStale AND this metadata header is present.
const metadataHeaderHeartbeatEnabled = "x-google-lock-file-uses-heartbeating"

var (
	// Time between consecutive heartbeats on the lock file.
	heartbeatInterval = 1 * time.Minute

	// The mininum duration that must have passed since the youngest
	// recorded heartbeat before the lock file is considered stale/orphaned.
	minHeartbeatAgeUntilStale = 15 * time.Minute
)

func (c *remoteClient) Get() (payload *remote.Payload, err error) {
	stateFileReader, err := c.stateFile().NewReader(c.storageContext)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, nil
		} else {
			return nil, fmt.Errorf("Failed to open state file at %v: %v", c.stateFileURL(), err)
		}
	}
	defer stateFileReader.Close()

	stateFileContents, err := ioutil.ReadAll(stateFileReader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file from %v: %v", c.stateFileURL(), err)
	}

	stateFileAttrs, err := c.stateFile().Attrs(c.storageContext)
	if err != nil {
		return nil, fmt.Errorf("Failed to read state file attrs from %v: %v", c.stateFileURL(), err)
	}

	result := &remote.Payload{
		Data: stateFileContents,
		MD5:  stateFileAttrs.MD5,
	}

	return result, nil
}

func (c *remoteClient) Put(data []byte) error {
	err := func() error {
		stateFileWriter := c.stateFile().NewWriter(c.storageContext)
		if _, err := stateFileWriter.Write(data); err != nil {
			return err
		}
		return stateFileWriter.Close()
	}()
	if err != nil {
		return fmt.Errorf("Failed to upload state to %v: %v", c.stateFileURL(), err)
	}

	return nil
}

func (c *remoteClient) Delete() error {
	if err := c.stateFile().Delete(c.storageContext); err != nil {
		return fmt.Errorf("Failed to delete state file %v: %v", c.stateFileURL(), err)
	}

	return nil
}

// createLockFile writes to a lock file, ensuring file creation. Returns the
// generation number.
func (c *remoteClient) createLockFile(lockFile *storage.ObjectHandle, infoJson []byte) (int64, error) {
	w := lockFile.If(storage.Conditions{DoesNotExist: true}).NewWriter(c.storageContext)
	err := func() error {
		if _, err := w.Write(infoJson); err != nil {
			return err
		}
		return w.Close()
	}()

	if err != nil {
		return 0, c.lockError(fmt.Errorf("writing %q failed: %v", c.lockFileURL(), err))
	}

	// Add metadata signalling to other clients that heartbeats will be
	// performed on this lock file.
	uattrs := storage.ObjectAttrsToUpdate{Metadata: make(map[string]string)}
	uattrs.Metadata[metadataHeaderHeartbeatEnabled] = "true"
	if _, err := lockFile.Update(c.storageContext, uattrs); err != nil {
		return 0, c.lockError(err)
	}

	return w.Attrs().Generation, nil
}

func isHeartbeatEnabled(attrs *storage.ObjectAttrs) bool {
	if attrs.Metadata != nil {
		if val, ok := attrs.Metadata[metadataHeaderHeartbeatEnabled]; ok {
			if val == "true" {
				return true
			}
		}
	}

	return false
}

// unlockIfStale force-unlocks the lock file if it is stale. Returns true if a
// stale lock was removed (and therefore a retry makes sense), otherwise false.
func (c *remoteClient) unlockIfStale(lockFile *storage.ObjectHandle) bool {
	if attrs, err := lockFile.Attrs(c.storageContext); err == nil {
		if !isHeartbeatEnabled(attrs) {
			// Metadata header metadataHeaderHeartbeatEnabled is
			// not present, thus this lock file is owned by an
			// older client that does not perform heartbeats on the
			// lock file. Therefore, we cannot be sure whether the
			// lock file might be stale. Better don't force-unlock!
			log.Printf("[TRACE] Found existing lock file %s from an older client that does not perform heartbeats", c.lockFileURL())
			return false
		}
		age := time.Now().Sub(attrs.Updated)
		if age > minHeartbeatAgeUntilStale {
			log.Printf("[WARN] Existing lock file %s is considered stale, last heartbeat was %s ago", c.lockFileURL(), age)
			if err := c.Unlock(strconv.FormatInt(attrs.Generation, 10)); err != nil {
				log.Printf("[WARN] Failed to release stale lock: %s", err)
			} else {
				return true
			}
		}
	}

	return false
}

// heartbeatLockFile periodically updates the "updated" timestamp of the lock
// file until the lock is released in Unlock().
func (c *remoteClient) heartbeatLockFile() {
	log.Printf("[TRACE] Starting heartbeat on lock file %s, interval is %s", c.lockFileURL(), heartbeatInterval)

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	defer func() {
		c.mutex.Lock()
		c.stopHeartbeatCh = nil
		c.mutex.Unlock()
	}()

	for {
		select {
		case <-c.stopHeartbeatCh:
			log.Printf("[TRACE] Stopping heartbeat on lock file %s", c.lockFileURL())
			return
		case t := <-ticker.C:
			log.Printf("[TRACE] Performing heartbeat on lock file %s, tick %q", c.lockFileURL(), t)
			if attrs, err := c.lockFile().Attrs(c.storageContext); err != nil {
				log.Printf("[WARN] Failed to retrieve attributes of lock file %s: %s", c.lockFileURL(), err)
			} else {
				c.mutex.Lock()
				generation := *c.generation
				c.mutex.Unlock()

				if attrs.Generation != generation {
					// This is no longer the lock file that we started with. Stop heartbeating on it.
					log.Printf("[WARN] Stopping heartbeat on lock file %s as it changed underneath.", c.lockFileURL())
					return
				}

				// Update the "updated" timestamp by removing non-existent metadata.
				uattrs := storage.ObjectAttrsToUpdate{Metadata: make(map[string]string)}
				uattrs.Metadata["x-goog-meta-terraform-state-heartbeat"] = ""
				if _, err := c.lockFile().Update(c.storageContext, uattrs); err != nil {
					log.Printf("[WARN] Failed to perform heartbeat on lock file %s: %s", c.lockFileURL(), err)
				}
			}
		}
	}
}

// Lock writes to a lock file, ensuring file creation. Returns the generation
// number, which must be passed to Unlock().
func (c *remoteClient) Lock(info *state.LockInfo) (string, error) {
	// update the path we're using
	// we can't set the ID until the info is written
	info.Path = c.lockFileURL()

	infoJson, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	lockFile := c.lockFile()

	generation, err := c.createLockFile(lockFile, infoJson)
	if err != nil {
		if c.unlockIfStale(lockFile) {
			generation, err = c.createLockFile(lockFile, infoJson)
		}
	}
	if err != nil {
		return "", c.lockError(fmt.Errorf("failed to create lock file %q: %v", c.lockFileURL(), err))
	}

	info.ID = strconv.FormatInt(generation, 10)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.generation = &generation
	c.stopHeartbeatCh = make(chan bool)

	go c.heartbeatLockFile()

	return info.ID, nil
}

func (c *remoteClient) Unlock(id string) error {
	gen, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.stopHeartbeatCh != nil {
		log.Printf("[TRACE] Stopping heartbeat on lock file %s before removing the lock", c.lockFileURL())
		c.stopHeartbeatCh <- true
	}

	if err := c.lockFile().If(storage.Conditions{GenerationMatch: gen}).Delete(c.storageContext); err != nil {
		return c.lockError(err)
	}

	return nil
}

func (c *remoteClient) lockError(err error) *state.LockError {
	lockErr := &state.LockError{
		Err: err,
	}

	info, infoErr := c.lockInfo()
	if infoErr != nil {
		lockErr.Err = multierror.Append(lockErr.Err, infoErr)
	} else {
		lockErr.Info = info
	}
	return lockErr
}

// lockInfo reads the lock file, parses its contents and returns the parsed
// LockInfo struct.
func (c *remoteClient) lockInfo() (*state.LockInfo, error) {
	r, err := c.lockFile().NewReader(c.storageContext)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	rawData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	info := &state.LockInfo{}
	if err := json.Unmarshal(rawData, info); err != nil {
		return nil, err
	}

	// We use the Generation as the ID, so overwrite the ID in the json.
	// This can't be written into the Info, since the generation isn't known
	// until it's written.
	attrs, err := c.lockFile().Attrs(c.storageContext)
	if err != nil {
		return nil, err
	}
	info.ID = strconv.FormatInt(attrs.Generation, 10)

	return info, nil
}

func (c *remoteClient) stateFile() *storage.ObjectHandle {
	h := c.storageClient.Bucket(c.bucketName).Object(c.stateFilePath)
	if len(c.encryptionKey) > 0 {
		return h.Key(c.encryptionKey)
	}
	return h
}

func (c *remoteClient) stateFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.stateFilePath)
}

func (c *remoteClient) lockFile() *storage.ObjectHandle {
	return c.storageClient.Bucket(c.bucketName).Object(c.lockFilePath)
}

func (c *remoteClient) lockFileURL() string {
	return fmt.Sprintf("gs://%v/%v", c.bucketName, c.lockFilePath)
}
