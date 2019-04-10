package oss

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"

	"encoding/hex"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	"github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/pkg/errors"
	"log"
	"sync"
	"time"
)

// Store the last saved serial in tablestore with this suffix for consistency checks.
const (
	stateIDSuffix = "-md5"
	statePKValue  = "terraform-remote-state-lock"
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

type TableStorePrimaryKeyMeta struct {
	PKName string
	PKType string
}

type RemoteClient struct {
	ossClient            *oss.Client
	otsClient            *tablestore.TableStoreClient
	bucketName           string
	stateFile            string
	lockFile             string
	serverSideEncryption bool
	acl                  string
	info                 *state.LockInfo
	mu                   sync.Mutex
	otsTable             string
	otsTabkePK           TableStorePrimaryKeyMeta
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
		return fmt.Errorf("Error getting bucket: %#v", err)
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
			return fmt.Errorf("Failed to upload state %s: %#v", c.stateFile, err)
		}
	}

	sum := md5.Sum(data)
	if err := c.putMD5(sum[:]); err != nil {
		// if this errors out, we unfortunately have to error out altogether,
		// since the next Get will inevitably fail.
		return fmt.Errorf("Failed to store state MD5: %s", err)
	}
	return nil
}

func (c *RemoteClient) Delete() error {
	bucket, err := c.ossClient.Bucket(c.bucketName)
	if err != nil {
		return fmt.Errorf("Error getting bucket %s: %#v", c.bucketName, err)
	}

	log.Printf("[DEBUG] Deleting remote state from OSS: %#v", c.stateFile)

	if err := bucket.DeleteObject(c.stateFile); err != nil {
		return fmt.Errorf("Error deleting state %s: %#v", c.stateFile, err)
	}

	if err := c.deleteMD5(); err != nil {
		log.Printf("[WARN] Error deleting state MD5: %s", err)
	}
	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	if c.otsTable == "" {
		return "", nil
	}

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
					ColumnName: c.otsTabkePK.PKName,
					Value:      c.getPKValue(),
				},
			},
		},
		Columns: []tablestore.AttributeColumn{
			{
				ColumnName: "LockID",
				Value:      c.lockFile,
			},
			{
				ColumnName: "Info",
				Value:      string(info.Marshal()),
			},
		},
		Condition: &tablestore.RowCondition{
			RowExistenceExpectation: tablestore.RowExistenceExpectation_EXPECT_NOT_EXIST,
		},
	}

	log.Printf("[DEBUG] Recoring state lock in tablestore: %#v", putParams)

	_, err := c.otsClient.PutRow(&tablestore.PutRowRequest{
		PutRowChange: putParams,
	})
	if err != nil {
		log.Printf("[WARN] Error storing state lock in tablestore: %#v", err)
		lockInfo, infoErr := c.getLockInfo()
		if infoErr != nil {
			log.Printf("[WARN] Error getting lock info: %#v", err)
			err = multierror.Append(err, infoErr)
		}
		lockErr := &state.LockError{
			Err:  err,
			Info: lockInfo,
		}
		log.Printf("[WARN] state lock error: %#v", lockErr)
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
					ColumnName: c.otsTabkePK.PKName,
					Value:      c.getPKValue(),
				},
			},
		},
		ColumnsToGet: []string{"LockID", "Digest"},
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
					ColumnName: c.otsTabkePK.PKName,
					Value:      c.getPKValue(),
				},
			},
		},
		Columns: []tablestore.AttributeColumn{
			{
				ColumnName: "LockID",
				Value:      c.lockPath() + stateIDSuffix,
			},
			{
				ColumnName: "Digest",
				Value:      hex.EncodeToString(sum),
			},
		},
		Condition: &tablestore.RowCondition{
			RowExistenceExpectation: tablestore.RowExistenceExpectation_EXPECT_NOT_EXIST,
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
						ColumnName: c.otsTabkePK.PKName,
						Value:      c.getPKValue(),
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

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	getParams := &tablestore.SingleRowQueryCriteria{
		TableName: c.otsTable,
		PrimaryKey: &tablestore.PrimaryKey{
			PrimaryKeys: []*tablestore.PrimaryKeyColumn{
				{
					ColumnName: c.otsTabkePK.PKName,
					Value:      c.getPKValue(),
				},
			},
		},
		ColumnsToGet: []string{"LockID", "Info"},
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
	lockInfo := &state.LockInfo{}
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

	lockErr := &state.LockError{}

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
						ColumnName: c.otsTabkePK.PKName,
						Value:      c.getPKValue(),
					},
				},
			},
			Condition: &tablestore.RowCondition{
				RowExistenceExpectation: tablestore.RowExistenceExpectation_EXPECT_EXIST,
			},
		},
	}

	log.Printf("[DEBUG] Deleting state lock from tablestore: %#v", params)

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
		return nil, fmt.Errorf("Error getting bucket %s: %#v", c.bucketName, err)
	}

	if exist, err := bucket.IsObjectExist(c.stateFile); err != nil {
		return nil, fmt.Errorf("Estimating object %s is exist got an error: %#v", c.stateFile, err)
	} else if !exist {
		return nil, nil
	}

	var options []oss.Option
	output, err := bucket.GetObject(c.stateFile, options...)
	if err != nil {
		return nil, fmt.Errorf("Error getting object: %#v", err)
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
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

func (c *RemoteClient) getPKValue() (value interface{}) {
	value = statePKValue
	if c.otsTabkePK.PKType == "Integer" {
		value = hashcode.String(statePKValue)
	} else if c.otsTabkePK.PKType == "Binary" {
		value = stringToBin(statePKValue)
	}
	return
}

func stringToBin(s string) (binString string) {
	for _, c := range s {
		binString = fmt.Sprintf("%s%b", binString, c)
	}
	return
}

const errBadChecksumFmt = `state data in OSS does not have the expected content.

This may be caused by unusually long delays in OSS processing a previous state
update.  Please wait for a minute or two and try again. If this problem
persists, and neither OSS nor TableStore are experiencing an outage, you may need
to manually verify the remote state and update the Digest value stored in the
TableStore table to the following value: %x
`
