// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oss

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

const (
	// Store the last saved serial in tablestore with this suffix for consistency checks.
	stateIDSuffix = "-md5"

	pkName = "LockID"
)

var (
	// The amount of time we will retry a state waiting for it to match the
	// expected checksum.
	consistencyRetryTimeout = 10 * time.Second

	// delay when polling the state
	consistencyRetryPollInterval = 2 * time.Second
)

// test hook called when checksums don't match
var testChecksumHook func()

type RemoteClient struct {
	ossClient            *oss.Client
	otsClient            *tablestore.TableStoreClient
	bucketName           string
	stateFile            string
	lockFile             string
	serverSideEncryption bool
	acl                  string
	otsTable             string
}

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	deadline := time.Now().Add(consistencyRetryTimeout)

	// If we have a checksum, and the returned payload doesn't match, we retry
	// up until deadline.
	for {
		payload, err = c.getObj()
		if err != nil {
			return nil, err
		}

		// If the remote state was manually removed the payload will be nil,
		// but if there's still a digest entry for that state we will still try
		// to compare the MD5 below.
		var digest []byte
		if payload != nil {
			digest = payload.MD5
		}

		// verify that this state is what we expect
		if expected, err := c.getMD5(); err != nil {
			log.Printf("[WARN] failed to fetch state md5: %s", err)
		} else if len(expected) > 0 && !bytes.Equal(expected, digest) {
			log.Printf("[WARN] state md5 mismatch: expected '%x', got '%x'", expected, digest)

			if testChecksumHook != nil {
				testChecksumHook()
			}

			if time.Now().Before(deadline) {
				time.Sleep(consistencyRetryPollInterval)
				log.Println("[INFO] retrying OSS RemoteClient.Get...")
				continue
			}

			return nil, fmt.Errorf(errBadChecksumFmt, digest)
		}

		break
	}
	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return fmt.Errorf("error getting bucket: %#v", err)
	}

	body := bytes.NewReader(data)

	var options []oss.Option
	if c.acl != "" {
		options = append(options, oss.ACL(oss.ACLType(c.acl)))
	}
	options = append(options, oss.ContentType("application/json"))
	if c.serverSideEncryption {
		options = append(options, oss.ServerSideEncryption("AES256"))
	}
	options = append(options, oss.ContentLength(int64(len(data))))

	if body != nil {
		if err := bucket.PutObject(c.stateFile, body, options...); err != nil {
			return fmt.Errorf("failed to upload state %s: %#v", c.stateFile, err)
		}
	}

	sum := md5.Sum(data)
	if err := c.putMD5(sum[:]); err != nil {
		// if this errors out, we unfortunately have to error out altogether,
		// since the next Get will inevitably fail.
		return fmt.Errorf("failed to store state MD5: %s", err)
	}
	return nil
}

func (c *RemoteClient) Delete() error {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return fmt.Errorf("error getting bucket %s: %#v", c.bucketName, err)
	}

	log.Printf("[DEBUG] Deleting remote state from OSS: %#v", c.stateFile)

	if err := bucket.DeleteObject(c.stateFile); err != nil {
		return fmt.Errorf("error deleting state %s: %#v", c.stateFile, err)
	}

	if err := c.deleteMD5(); err != nil {
		log.Printf("[WARN] Error deleting state MD5: %s", err)
	}
	return nil
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	if c.otsTable == "" {
		return "", nil
	}

	info.Path = c.lockPath()

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		info.ID = lockID
	}

	putParams := &tablestore.PutRowChange{
		TableName: c.otsTable,
		PrimaryKey: &tablestore.PrimaryKey{
			PrimaryKeys: []*tablestore.PrimaryKeyColumn{
				{
					ColumnName: pkName,
					Value:      c.lockPath(),
				},
			},
		},
		Columns: []tablestore.AttributeColumn{
			{
				ColumnName: "Info",
				Value:      string(info.Marshal()),
			},
		},
		Condition: &tablestore.RowCondition{
			RowExistenceExpectation: tablestore.RowExistenceExpectation_EXPECT_NOT_EXIST,
		},
	}

	log.Printf("[DEBUG] Recording state lock in tablestore: %#v; LOCKID:%s", putParams, c.lockPath())

	_, err := c.otsClient.PutRow(&tablestore.PutRowRequest{
		PutRowChange: putParams,
	})
	if err != nil {
		err = fmt.Errorf("invoking PutRow got an error: %#v", err)
		lockInfo, infoErr := c.getLockInfo()
		if infoErr != nil {
			err = errors.Join(err, fmt.Errorf("\ngetting lock info got an error: %#v", infoErr))
		}
		lockErr := &statemgr.LockError{
			Err:  err,
			Info: lockInfo,
		}
		log.Printf("[ERROR] state lock error: %s", lockErr.Error())
		return "", lockErr
	}

	return info.ID, nil
}

func (c *RemoteClient) getMD5() ([]byte, error) {
	if c.otsTable == "" {
		return nil, nil
	}

	getParams := &tablestore.SingleRowQueryCriteria{
		TableName: c.otsTable,
		PrimaryKey: &tablestore.PrimaryKey{
			PrimaryKeys: []*tablestore.PrimaryKeyColumn{
				{
					ColumnName: pkName,
					Value:      c.lockPath() + stateIDSuffix,
				},
			},
		},
		ColumnsToGet: []string{pkName, "Digest"},
		MaxVersion:   1,
	}

	log.Printf("[DEBUG] Retrieving state serial in tablestore: %#v", getParams)

	object, err := c.otsClient.GetRow(&tablestore.GetRowRequest{
		SingleRowQueryCriteria: getParams,
	})

	if err != nil {
		return nil, err
	}

	var val string
	if v, ok := object.GetColumnMap().Columns["Digest"]; ok && len(v) > 0 {
		val = v[0].Value.(string)
	}

	sum, err := hex.DecodeString(val)
	if err != nil || len(sum) != md5.Size {
		return nil, errors.New("invalid md5")
	}

	return sum, nil
}

// store the hash of the state to that clients can check for stale state files.
func (c *RemoteClient) putMD5(sum []byte) error {
	if c.otsTable == "" {
		return nil
	}

	if len(sum) != md5.Size {
		return errors.New("invalid payload md5")
	}

	putParams := &tablestore.PutRowChange{
		TableName: c.otsTable,
		PrimaryKey: &tablestore.PrimaryKey{
			PrimaryKeys: []*tablestore.PrimaryKeyColumn{
				{
					ColumnName: pkName,
					Value:      c.lockPath() + stateIDSuffix,
				},
			},
		},
		Columns: []tablestore.AttributeColumn{
			{
				ColumnName: "Digest",
				Value:      hex.EncodeToString(sum),
			},
		},
		Condition: &tablestore.RowCondition{
			RowExistenceExpectation: tablestore.RowExistenceExpectation_IGNORE,
		},
	}

	log.Printf("[DEBUG] Recoring state serial in tablestore: %#v", putParams)

	_, err := c.otsClient.PutRow(&tablestore.PutRowRequest{
		PutRowChange: putParams,
	})

	if err != nil {
		log.Printf("[WARN] failed to record state serial in tablestore: %s", err)
	}

	return nil
}

// remove the hash value for a deleted state
func (c *RemoteClient) deleteMD5() error {
	if c.otsTable == "" {
		return nil
	}

	params := &tablestore.DeleteRowRequest{
		DeleteRowChange: &tablestore.DeleteRowChange{
			TableName: c.otsTable,
			PrimaryKey: &tablestore.PrimaryKey{
				PrimaryKeys: []*tablestore.PrimaryKeyColumn{
					{
						ColumnName: pkName,
						Value:      c.lockPath() + stateIDSuffix,
					},
				},
			},
			Condition: &tablestore.RowCondition{
				RowExistenceExpectation: tablestore.RowExistenceExpectation_EXPECT_EXIST,
			},
		},
	}

	log.Printf("[DEBUG] Deleting state serial in tablestore: %#v", params)

	if _, err := c.otsClient.DeleteRow(params); err != nil {
		return err
	}

	return nil
}

func (c *RemoteClient) getLockInfo() (*statemgr.LockInfo, error) {
	getParams := &tablestore.SingleRowQueryCriteria{
		TableName: c.otsTable,
		PrimaryKey: &tablestore.PrimaryKey{
			PrimaryKeys: []*tablestore.PrimaryKeyColumn{
				{
					ColumnName: pkName,
					Value:      c.lockPath(),
				},
			},
		},
		ColumnsToGet: []string{pkName, "Info"},
		MaxVersion:   1,
	}

	log.Printf("[DEBUG] Retrieving state lock info from tablestore: %#v", getParams)

	object, err := c.otsClient.GetRow(&tablestore.GetRowRequest{
		SingleRowQueryCriteria: getParams,
	})
	if err != nil {
		return nil, err
	}

	var infoData string
	if v, ok := object.GetColumnMap().Columns["Info"]; ok && len(v) > 0 {
		infoData = v[0].Value.(string)
	}
	lockInfo := &statemgr.LockInfo{}
	err = json.Unmarshal([]byte(infoData), lockInfo)
	if err != nil {
		return nil, err
	}
	return lockInfo, nil
}
func (c *RemoteClient) Unlock(id string) error {
	if c.otsTable == "" {
		return nil
	}

	lockErr := &statemgr.LockError{}

	lockInfo, err := c.getLockInfo()
	if err != nil {
		lockErr.Err = fmt.Errorf("failed to retrieve lock info: %s", err)
		return lockErr
	}
	lockErr.Info = lockInfo

	if lockInfo.ID != id {
		lockErr.Err = fmt.Errorf("lock id %q does not match existing lock", id)
		return lockErr
	}
	params := &tablestore.DeleteRowRequest{
		DeleteRowChange: &tablestore.DeleteRowChange{
			TableName: c.otsTable,
			PrimaryKey: &tablestore.PrimaryKey{
				PrimaryKeys: []*tablestore.PrimaryKeyColumn{
					{
						ColumnName: pkName,
						Value:      c.lockPath(),
					},
				},
			},
			Condition: &tablestore.RowCondition{
				RowExistenceExpectation: tablestore.RowExistenceExpectation_IGNORE,
			},
		},
	}

	_, err = c.otsClient.DeleteRow(params)

	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	return nil
}

func (c *RemoteClient) lockPath() string {
	return fmt.Sprintf("%s/%s", c.bucketName, c.stateFile)
}

func (c *RemoteClient) getObj() (*remote.Payload, error) {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return nil, fmt.Errorf("error getting bucket %s: %#v", c.bucketName, err)
	}

	if exist, err := bucket.IsObjectExist(c.stateFile); err != nil {
		return nil, fmt.Errorf("estimating object %s is exist got an error: %#v", c.stateFile, err)
	} else if !exist {
		return nil, nil
	}

	var options []oss.Option
	output, err := bucket.GetObject(c.stateFile, options...)
	if err != nil {
		return nil, fmt.Errorf("error getting object: %#v", err)
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output); err != nil {
		return nil, fmt.Errorf("failed to read remote state: %s", err)
	}
	sum := md5.Sum(buf.Bytes())
	payload := &remote.Payload{
		Data: buf.Bytes(),
		MD5:  sum[:],
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

const errBadChecksumFmt = `state data in OSS does not have the expected content.

This may be caused by unusually long delays in OSS processing a previous state
update.  Please wait for a minute or two and try again. If this problem
persists, and neither OSS nor TableStore are experiencing an outage, you may need
to manually verify the remote state and update the Digest value stored in the
TableStore table to the following value: %x`
