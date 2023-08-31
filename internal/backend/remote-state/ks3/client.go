package ks3

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/KscSDK/ksc-sdk-go/service/tagv2"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/wilac-pv/ksyun-ks3-go-sdk/ks3"
)

var emptyTime = time.Time{}

type remoteClient struct {
	ks3Context context.Context

	ks3Client *ks3.Client
	tagClient *tagv2.Tagv2

	bucketName   string
	bucket       *ks3.Bucket
	stateFile    string
	lockFile     string
	lockDuration time.Duration

	encrypt bool
	acl     string
}

const (
	KsyunLockTagKey = "ksyun-terraform-lock"

	TagExistsErr = "TagAlreadyExistsLimitExceeded"
)

func (c *remoteClient) Put(data []byte) error {
	log.Printf("[DEBUG] upload state file to ksyun remote")

	return c.putObject(c.stateFile, data)
}

func (c *remoteClient) Get() (*remote.Payload, error) {
	log.Printf("[DEBUG] get remote state file %s", c.stateFile)

	exists, data, checksum, err := c.getObject(c.stateFile)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	payload := &remote.Payload{
		Data: data,
		MD5:  []byte(checksum),
	}

	return payload, nil
}

// Delete will remove remote state file
func (c *remoteClient) Delete() error {
	log.Printf("[DEBUG] delete remote state file: %s", c.stateFile)

	return c.deleteObject(c.stateFile)
}

func (c *remoteClient) Lock(lockInfo *statemgr.LockInfo) (string, error) {
	log.Printf("[DEBUG] lock remote state file %s", c.lockFile)
	err := c.ks3Lock(lockInfo)
	if err != nil {
		return "", c.lockError(err)
	}
	defer c.ks3Unlock(lockInfo)

	exist, existData, _, err := c.getObject(c.lockFile)
	if exist {
		existLock := &statemgr.LockInfo{}
		if parseErr := json.Unmarshal(existData, existLock); parseErr != nil {
			return "", c.lockError(fmt.Errorf("unmarshal exist lock file error, %s", parseErr))
		}
		if !isKs3lockBeyondTime(existLock.Created, c.lockDuration) {
			return "", c.lockError(fmt.Errorf("lock file %s exists", c.lockFile))
		}
	}

	lockInfo.Path = c.lockFile

	data, err := json.Marshal(lockInfo)
	if err != nil {
		return "", c.lockError(err)
	}

	check := fmt.Sprintf("%x", md5.Sum(data))
	// write to lock file that's the lock entity
	if err := c.putObject(c.lockFile, data); err != nil {
		return "", c.lockError(err)
	}

	return check, nil
}

func (c *remoteClient) Unlock(check string) error {
	log.Printf("[DEBUG] unlock remote state file %s\n", c.lockFile)

	lockInfo, err := c.lockInfo()
	if err != nil {
		return err
	}

	if lockInfo.ID != check {
		return fmt.Errorf("lock id mismatch, %v != %v", lockInfo.ID, check)
	}

	err = c.deleteObject(c.lockFile)
	if err != nil {
		return err
	}

	err = c.ks3Unlock(lockInfo)
	if err != nil {
		return err
	}
	return nil
}

func (c *remoteClient) ks3Lock(tfLock *statemgr.LockInfo) error {
	ks3LockValue := c.ks3LockValueStr(tfLock)

	tagExist, createTime, err := c.CheckTag(KsyunLockTagKey, ks3LockValue)
	if err != nil {
		return fmt.Errorf("an error caused by check ks3lock: %s", err)
	}

	// if ks3lock was existed and beyond c.lockDuration, delete the old ks3lock
	if tagExist && isKs3lockBeyondTime(createTime, c.lockDuration) {
		err := c.DeleteTag(KsyunLockTagKey, ks3LockValue)
		if err != nil {
			return fmt.Errorf("fail to clean the old ks3lock, %s -> %s, ks3lock create time: %s, err: %s", KsyunLockTagKey, ks3LockValue, createTime.String(), err)
		}
	}

	err = c.CreateTag(KsyunLockTagKey, ks3LockValue)
	if err != nil {
		if kscErr, ok := err.(awserr.Error); ok {
			if kscErr.Code() == TagExistsErr {
				return fmt.Errorf("ks3Lock is exist, %s -> %s, %s", KsyunLockTagKey, ks3LockValue, err)
			}
			return err
		}
	}
	return err
}

func (c *remoteClient) ks3Unlock(tfLock *statemgr.LockInfo) error {
	ks3LockValue := c.ks3LockValueStr(tfLock)
	ks3LockKey := KsyunLockTagKey

	var err error
	for i := 0; i < 30; i++ {
		tagExists, _, err := c.CheckTag(ks3LockKey, ks3LockValue)

		if err != nil {
			return err
		}

		if !tagExists {
			return nil
		}

		err = c.DeleteTag(ks3LockKey, ks3LockValue)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return err
}

// lockError returns statemgr.LockError
func (c *remoteClient) lockError(err error) *statemgr.LockError {
	log.Printf("[DEBUG] failed to lock or unlock %s: %v", c.lockFile, err)

	lockErr := &statemgr.LockError{
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

// lockInfo returns LockInfo from lock file
func (c *remoteClient) lockInfo() (*statemgr.LockInfo, error) {
	exist, data, check, err := c.getObject(c.lockFile)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, fmt.Errorf("lock file %s not exists", c.lockFile)
	}

	info := &statemgr.LockInfo{}
	if err := json.Unmarshal(data, info); err != nil {
		return nil, err
	}

	info.ID = check

	return info, nil
}

func (c *remoteClient) CreateTag(key, value string) error {
	req := map[string]interface{}{}

	req["Key"] = key
	req["Value"] = value
	if _, err := c.tagClient.CreateTag(&req); err != nil {
		return err
	}
	return nil

}

func (c *remoteClient) CheckTag(key, value string) (exists bool, createTime time.Time, err error) {
	req := map[string]interface{}{}
	createTime = emptyTime
	req["Key"] = key
	req["Value"] = value
	resp, err := c.tagClient.ListTags(&req)
	if err != nil {
		return exists, createTime, err
	}
	tagsIf, ok := (*resp)["Tags"]
	if !ok {
		return
	}

	if tags, ok := tagsIf.([]interface{}); ok && len(tags) > 0 {
		tag := tags[0].(map[string]interface{})
		tagKey := tag["Key"].(string)
		tagValue := tag["Value"].(string)
		createTimeStr := tag["CreateTime"].(string)
		formatTime, parseErr := time.Parse(time.DateTime, createTimeStr)
		if parseErr != nil {
			err = parseErr
			return
		}
		createTime = formatTime
		exists = key == tagKey && value == tagValue

	}
	return

}

func (c *remoteClient) DeleteTag(key, value string) error {
	params := make(map[string]interface{}, 1)
	tagMap := map[string]string{
		"Key":   key,
		"Value": value,
	}
	params["Tags"] = []interface{}{tagMap}
	_, err := c.tagClient.DeleteTag(&params)
	if err != nil {
		return err
	}
	return nil
}

func (c *remoteClient) listObjects(prefix string) ([]ks3.ObjectProperties, error) {
	objectsResult, err := c.bucket.ListObjectsV2(ks3.Prefix(prefix))
	if err != nil {
		return nil, err
	}
	return objectsResult.Objects, nil
}

func (c *remoteClient) deleteObject(objectName string) error {
	err := c.bucket.DeleteObject(objectName)
	if err != nil {
		log.Printf("[DEBUG] deleteObject %s: error: %v", objectName, err)
		return fmt.Errorf("failed to remove object %s: %s", objectName, err)
	}
	return nil
}
func (c *remoteClient) putObject(objectName string, data []byte) error {
	var build strings.Builder

	build.Write(data)
	objectReader := strings.NewReader(build.String())

	var options []ks3.Option
	optionsAcl := ks3.ObjectACL(ks3.ACLType(c.acl))

	options = append(options, optionsAcl)
	if c.encrypt {
		encryption := ks3.ServerSideEncryption("AES256")
		options = append(options, encryption)
	}

	// upload file content
	err := c.bucket.PutObject(objectName, objectReader, options...)
	if err != nil {
		log.Printf("[DEBUG] failed to upload object: %s, error: %v", objectName, err)
		return fmt.Errorf("fatil to upload state file: %s, error: %v", objectName, err)
	}
	return nil
}

func (c *remoteClient) getObject(objectName string) (exist bool, data []byte, check string, err error) {
	resultReader, err := c.bucket.GetObject(objectName)
	if err != nil {
		if ks3Err, ok := err.(ks3.ServiceError); ok {
			if ks3Err.StatusCode == 404 {
				exist = false
				err = nil
			}
		} else {
			err = fmt.Errorf("failed to get object %s, error: %v", objectName, err)
		}
		log.Printf("[DEBUG] failed to get object content %s, %v", objectName, err)

		return
	}
	defer resultReader.Close()

	exist = true
	data, err = io.ReadAll(resultReader)
	if err != nil {
		log.Printf("[DEBUG] failed to read object content in local, err: %v", err)
		err = fmt.Errorf("failed to read object content in local, err: %v", err)
		return
	}
	log.Printf("[DEBUG] read object %s data length: %d", objectName, len(data))

	check = fmt.Sprintf("%x", md5.Sum(data))
	return exist, data, check, err
}

func (c *remoteClient) putBucket() error {
	log.Printf("[DEBUG] create transient bucket")

	// for testing case, so set directly public read and write.
	bucketAcl := ks3.ACL(ks3.ACLPublicReadWrite)
	if err := c.ks3Client.CreateBucket(c.bucketName, bucketAcl); err != nil {
		if ks3Err, ok := err.(ks3.ServiceError); ok && ks3Err.StatusCode == 409 {
			return nil
		}
		return err
	}
	return nil
}

func (c *remoteClient) deleteBucket(emptyBucket bool) error {
	if emptyBucket {
		obs, err := c.listObjects("")
		if err != nil {
			if strings.Contains(err.Error(), "not exists") {
				return nil
			}
			log.Printf("[DEBUG] deleteBucket %s: empty bucket error: %v", c.bucketName, err)
			return fmt.Errorf("failed to empty bucket %v: %v", c.bucketName, err)
		}
		for _, v := range obs {
			err := c.deleteObject(v.Key)
			if err != nil {
				return err
			}
		}
	}

	err := c.ks3Client.DeleteBucket(c.bucketName)
	if err != nil {
		if ks3Err, ok := err.(ks3.ServiceError); ok && ks3Err.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("failed to delete bucket %v: %v", c.bucketName, err)
	}

	return nil
}

func (c *remoteClient) ks3LockValueStr(lockInfo *statemgr.LockInfo) string {
	ret := strings.Join([]string{c.bucketName, c.lockFile}, ":")
	return ret
}

func isKs3lockBeyondTime(cTime time.Time, between time.Duration) bool {
	if cTime == emptyTime {
		return false
	}

	now := time.Now()

	if now.Sub(cTime).Abs() > between {
		return true
	}
	return false
}
