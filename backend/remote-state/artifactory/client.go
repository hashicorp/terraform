package artifactory

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	artifactory "github.com/lusis/go-artifactory/src/artifactory.v401"
)

const ARTIF_TFSTATE_NAME = "terraform.tfstate"
const ARTIF_TFLOCK_NAME = "terraform.lock"

type ArtifactoryClient struct {
	nativeClient       *artifactory.ArtifactoryClient
	lockNativeClient   *artifactory.ArtifactoryClient
	unlockNativeClient *artifactory.ArtifactoryClient
	userName           string
	password           string
	url                string
	repo               string
	subpath            string
	lockUserName       string
	lockPassword       string
	unlockUserName     string
	unlockPassword     string
	lockUrl            string
	lockRepo           string
	lockSubpath        string
	lockID             string
	jsonLockInfo       []byte
	lockReadbackWait   int
}

func logStart(action string) {
	log.Printf("[TRACE] backend/remote-state/artifactory: starting %s operation", action)
}

func logResult(action string, err *error) {
	if *err == nil {
		log.Printf("[TRACE] backend/remote-state/artifactory: exiting %s operation with success", action)
	} else {
		log.Printf("[TRACE] backend/remote-state/artifactory: exiting %s operation with failure", action)
	}
}
func (c *ArtifactoryClient) Get() (*remote.Payload, error) {
	var err error
	logStart("Get")
	defer logResult("Get", &err)
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	output, err := c.nativeClient.Get(p, make(map[string]string))
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}

	hash := md5.Sum(output)
	payload := &remote.Payload{
		Data: output,
		MD5:  hash[:md5.Size],
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *ArtifactoryClient) Put(data []byte) error {
	var err error
	logStart("Put")
	defer logResult("Put", &err)

	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	if _, err = c.nativeClient.Put(p, string(data), make(map[string]string)); err == nil {
		return nil
	} else {
		return fmt.Errorf("Failed to upload state: %v", err)
	}
}

func (c *ArtifactoryClient) Delete() error {
	var err error
	logStart("Delete")
	defer logResult("Delete", &err)
	p := fmt.Sprintf("%s/%s/%s", c.repo, c.subpath, ARTIF_TFSTATE_NAME)
	err = c.nativeClient.Delete(p)
	return err
}

func (c *ArtifactoryClient) Lock(info *state.LockInfo) (string, error) {
	if c.lockUrl == "" {
		return "", nil
	}
	c.lockID = ""
	var err error
	var output []byte
	logStart("Lock")
	defer logResult("Lock", &err)
	rand.Seed(time.Now().UnixNano())
	info.ID = fmt.Sprintf("%s-%d", info.ID, rand.Intn(100000000))
	jsonLockInfo := info.Marshal()
	p := fmt.Sprintf("%s/%s/%s", c.lockRepo, c.lockSubpath, ARTIF_TFLOCK_NAME)
	if _, err = c.lockNativeClient.Put(p, string(jsonLockInfo), make(map[string]string)); err == nil {
		if c.lockReadbackWait <= 0 {
			c.lockID = info.ID
			c.jsonLockInfo = jsonLockInfo
			return info.ID, nil
		}
		time.Sleep(time.Duration(c.lockReadbackWait) * time.Millisecond)
		// readback and compare with original info
		output, err = c.lockNativeClient.Get(p, make(map[string]string))
		if err != nil {
			return "", fmt.Errorf("Failed to read back lockInfo. Failed to lock: %v", err)
		} else {
			var lockInfoReadBack state.LockInfo
			err = json.Unmarshal(output, &lockInfoReadBack)
			if err != nil {
				return "", fmt.Errorf("Failed to Unmarchal lockInfo from readback. Failed to lock: %v", err)
			} else {
				if reflect.DeepEqual(*info, lockInfoReadBack) {
					c.lockID = info.ID
					c.jsonLockInfo = jsonLockInfo
					return info.ID, nil
				} else {
					err = errors.New("lockinfo readback and original are not the same. Failed to lock")
					return "", err
				}
			}
		}
	} else {
		return "", fmt.Errorf("Failed to lock: %v", err)
	}
}

func (c *ArtifactoryClient) Unlock(id string) error {
	if c.lockUrl == "" {
		return nil
	}
	var err error
	logStart("Unlock")
	defer logResult("Unlock", &err)
	p := fmt.Sprintf("%s/%s/%s", c.lockRepo, c.lockSubpath, ARTIF_TFLOCK_NAME)
	if c.unlockUserName == "" {
		err = c.lockNativeClient.Delete(p)
	} else {
		err = c.unlockNativeClient.Delete(p)
	}
	if err != nil {
		log.Printf("[WARN] backend/remote-state/artifactory: failed to unlock")
		return fmt.Errorf("Failed to unlock state: %v", err)
	}
	return err
}
