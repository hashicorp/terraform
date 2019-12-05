package dynamodb

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	multierror "github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Store the last saved serial in dynamo with this suffix for consistency checks.
const (
	stateIDSuffix    = "-md5"
	dynamoDBItemSize = 400000
)

type RemoteClient struct {
	dynClient *dynamodb.DynamoDB
	tableName string
	path      string
	lockTable string
}

var (
	// The amount of time we will retry a state waiting for it to match the expected checksum.
	consistencyRetryTimeout = 10 * time.Second

	// delay when polling the state
	consistencyRetryPollInterval = 2 * time.Second
)

// test hook called when checksums don't match
var testChecksumHook func()

func (c *RemoteClient) Get() (payload *remote.Payload, err error) {
	deadline := time.Now().Add(consistencyRetryTimeout)

	// If we have a checksum, and the returned payload doesn't match, we retry
	// up until deadline.
	for {
		payload, err = c.get()
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
				log.Println("[INFO] retrying S3 RemoteClient.Get...")
				continue
			}

			return nil, fmt.Errorf(errBadChecksumFmt, digest)
		}

		break
	}

	return payload, err
}

func getMaxSegmentId(items []map[string]*dynamodb.AttributeValue) (int, error) {
	maxSegmentID := 0
	for _, i := range items {
		state := State{}
		err := dynamodbattribute.UnmarshalMap(i, &state)
		if err != nil {
			return -1, fmt.Errorf("Got error marshalling state: %s", err)
		}
		segmentID, err := strconv.Atoi(state.SegmentID)
		if err != nil {
			return -1, fmt.Errorf("Got error casting: %s", err)
		}
		if segmentID > maxSegmentID {
			maxSegmentID = segmentID
		}
	}
	return maxSegmentID, nil
}

func (c *RemoteClient) get() (*remote.Payload, error) {
	var queryInput = &dynamodb.QueryInput{
		TableName: aws.String(c.tableName),
		KeyConditions: map[string]*dynamodb.Condition{
			"StateID": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(c.path),
					},
				},
			},
		},
	}

	result, err := c.dynClient.Query(queryInput)
	if err != nil {
		return nil, fmt.Errorf("During query operation on table %s %s.", c.tableName, err)
	}

	maxSegmentID, err := getMaxSegmentId(result.Items)
	if err != nil {
		return nil, err
	}
	var segmentStrings = make([]string, maxSegmentID+1)

	for _, i := range result.Items {
		state := State{}
		err = dynamodbattribute.UnmarshalMap(i, &state)
		if err != nil {
			return nil, fmt.Errorf("Got error marshalling state: %s", err)
		}
		segmentID, err := strconv.Atoi(state.SegmentID)
		if err != nil {
			return nil, fmt.Errorf("Got error casting: %s", err)
		}
		segmentStrings[segmentID] = state.Body
	}

	jsonString := strings.Join(segmentStrings[:], "")
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, strings.NewReader(jsonString)); err != nil {
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

func (c *RemoteClient) GeneratePutItems(data []byte, sequence []int, transactionItems *[]*dynamodb.TransactWriteItem) error {
	body := string(data[:])

	item := State{
		StateID:   c.path,
		SegmentID: strconv.Itoa(sequence[0]),
		Body:      body,
	}

	b, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("Got error marshalling item: %s", err)
	}

	if len(b) < dynamoDBItemSize {
		av, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
			return fmt.Errorf("Got error marshalling state: %s", err)
		}

		put_item := &dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: aws.String(c.tableName),
				Item:      av,
			},
		}

		*transactionItems = append(*transactionItems, put_item)

	} else {
		N := int(len(data) / 2)
		err := c.GeneratePutItems(data[N:], sequence[N:], transactionItems)
		if err != nil {
			return fmt.Errorf("Got error during put generation: %s", err)
		}
		err = c.GeneratePutItems(data[:N], sequence[:N], transactionItems)
		if err != nil {
			return fmt.Errorf("Got error during put generation: %s", err)
		}
	}

	return nil
}

func GenerateSequence(sequenceSize int, currentSegments []int) []int {
	if sequenceSize == 0 {
		return []int{0}
	}

	segmentsSize := len(currentSegments)
	sequence := make([]int, sequenceSize)
	position := 0
	for index := 0; index < sequenceSize+segmentsSize; index++ {
		to_use := true
		for _, segment := range currentSegments {
			to_use = !(segment == index) && to_use
		}
		if to_use {
			sequence[position] = index
			position += 1
		}
	}
	return sequence
}

func (c *RemoteClient) Put(data []byte) error {
	var queryInput = &dynamodb.QueryInput{
		TableName: aws.String(c.tableName),
		KeyConditions: map[string]*dynamodb.Condition{
			"StateID": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(c.path),
					},
				},
			},
		},
	}

	result, err := c.dynClient.Query(queryInput)
	if err != nil {
		return fmt.Errorf("During query operation on table %s %s.", c.tableName, err)
	}
	var transactionItems = make([]*dynamodb.TransactWriteItem, 0)
	var segments []int
	for _, i := range result.Items {
		state := State{}

		err = dynamodbattribute.UnmarshalMap(i, &state)
		if err != nil {
			return fmt.Errorf("Got error marshalling state: %s", err)
		}

		delete_item := &dynamodb.TransactWriteItem{
			Delete: &dynamodb.Delete{
				TableName: aws.String(c.tableName),
				Key: map[string]*dynamodb.AttributeValue{
					"StateID": {
						S: aws.String(state.StateID),
					},
					"SegmentID": {
						S: aws.String(state.SegmentID),
					},
				},
			},
		}
		transactionItems = append(transactionItems, delete_item)
		id, err := strconv.Atoi(state.SegmentID)
		if err != nil {
			return fmt.Errorf("Got error casting: %s", err)
		}
		segments = append(segments, id)
	}

	sequence := GenerateSequence(len(data), segments)
	log.Printf("[DEBUG] Uploading remote state to DynamoDB: %#v", transactionItems)

	err = c.GeneratePutItems(data, sequence, &transactionItems)
	if err != nil {
		return fmt.Errorf("Got error calling GeneratePutItems: %s", err)
	}

	_, err = c.dynClient.TransactWriteItems(&dynamodb.TransactWriteItemsInput{TransactItems: transactionItems})
	if err != nil {
		return fmt.Errorf("Got error calling TransactWriteItems: %s", err)
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
	var queryInput = &dynamodb.QueryInput{
		TableName: aws.String(c.tableName),
		KeyConditions: map[string]*dynamodb.Condition{
			"StateID": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(c.path),
					},
				},
			},
		},
	}

	result, err := c.dynClient.Query(queryInput)
	if err != nil {
		return err
	}
	var transactionItems = make([]*dynamodb.TransactWriteItem, 0)
	for _, i := range result.Items {
		state := State{}

		err = dynamodbattribute.UnmarshalMap(i, &state)
		if err != nil {
			return fmt.Errorf("Got error marshalling state: %s", err)
		}
		delete_item := &dynamodb.TransactWriteItem{
			Delete: &dynamodb.Delete{
				TableName: aws.String(c.tableName),
				Key: map[string]*dynamodb.AttributeValue{
					"StateID": {
						S: aws.String(state.StateID),
					},
					"SegmentID": {
						S: aws.String(state.SegmentID),
					},
				},
			},
		}
		transactionItems = append(transactionItems, delete_item)
	}

	_, err = c.dynClient.TransactWriteItems(&dynamodb.TransactWriteItemsInput{TransactItems: transactionItems})
	if err != nil {
		return fmt.Errorf("Got error calling TransactWriteItems: %s", err)
	}

	if err != nil {
		return err
	}

	if err := c.deleteMD5(); err != nil {
		log.Printf("Error deleting state md5: %s", err)
	}

	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	if c.lockTable == "" {
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

	putParams := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath())},
			"Info":   {S: aws.String(string(info.Marshal()))},
		},
		TableName:           aws.String(c.lockTable),
		ConditionExpression: aws.String("attribute_not_exists(LockID)"),
	}
	_, err := c.dynClient.PutItem(putParams)

	if err != nil {
		lockInfo, infoErr := c.getLockInfo()
		if infoErr != nil {
			err = multierror.Append(err, infoErr)
		}

		lockErr := &state.LockError{
			Err:  err,
			Info: lockInfo,
		}
		return "", lockErr
	}

	return info.ID, nil
}

func (c *RemoteClient) getMD5() ([]byte, error) {
	if c.lockTable == "" {
		return nil, nil
	}

	getParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath() + stateIDSuffix)},
		},
		ProjectionExpression: aws.String("LockID, Digest"),
		TableName:            aws.String(c.lockTable),
		ConsistentRead:       aws.Bool(true),
	}

	resp, err := c.dynClient.GetItem(getParams)
	if err != nil {
		return nil, err
	}

	var val string
	if v, ok := resp.Item["Digest"]; ok && v.S != nil {
		val = *v.S
	}

	sum, err := hex.DecodeString(val)
	if err != nil || len(sum) != md5.Size {
		return nil, errors.New("invalid md5")
	}

	return sum, nil
}

// store the hash of the state so that clients can check for stale state files.
func (c *RemoteClient) putMD5(sum []byte) error {
	if c.lockTable == "" {
		return nil
	}

	if len(sum) != md5.Size {
		return errors.New("invalid payload md5")
	}

	putParams := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath() + stateIDSuffix)},
			"Digest": {S: aws.String(hex.EncodeToString(sum))},
		},
		TableName: aws.String(c.lockTable),
	}
	_, err := c.dynClient.PutItem(putParams)
	if err != nil {
		log.Printf("[WARN] failed to record state serial in dynamodb: %s", err)
	}

	return nil
}

// remove the hash value for a deleted state
func (c *RemoteClient) deleteMD5() error {
	if c.lockTable == "" {
		return nil
	}

	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath() + stateIDSuffix)},
		},
		TableName: aws.String(c.lockTable),
	}
	if _, err := c.dynClient.DeleteItem(params); err != nil {
		return err
	}
	return nil
}

func (c *RemoteClient) getLockInfo() (*state.LockInfo, error) {
	getParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath())},
		},
		ProjectionExpression: aws.String("LockID, Info"),
		TableName:            aws.String(c.lockTable),
		ConsistentRead:       aws.Bool(true),
	}

	resp, err := c.dynClient.GetItem(getParams)
	if err != nil {
		return nil, err
	}

	var infoData string
	if v, ok := resp.Item["Info"]; ok && v.S != nil {
		infoData = *v.S
	}

	lockInfo := &state.LockInfo{}
	err = json.Unmarshal([]byte(infoData), lockInfo)
	if err != nil {
		return nil, err
	}

	return lockInfo, nil
}

func (c *RemoteClient) Unlock(id string) error {
	if c.lockTable == "" {
		return nil
	}

	lockErr := &state.LockError{}

	// TODO: store the path and lock ID in separate fields, and have proper
	// projection expression only delete the lock if both match, rather than
	// checking the ID from the info field first.
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

	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath())},
		},
		TableName: aws.String(c.lockTable),
	}
	_, err = c.dynClient.DeleteItem(params)

	if err != nil {
		lockErr.Err = err
		return lockErr
	}
	return nil
}

func (c *RemoteClient) lockPath() string {
	return fmt.Sprintf("%s/%s", c.tableName, c.path)
}

const errBadChecksumFmt = `State data in DynamoDB does not have the expected content.

This may be caused by unusually long delays in DynamoDB processing a previous state
update.  Please wait for a minute or two and try again. If this problem
persists, and DynamoDB is not experiencing an outage, you may need
to manually verify the remote state and update the Digest value stored in the
DynamoDB lock table to the following value: %x
`

const errS3NoSuchBucket = `DynamoDB table does not exist.

The referenced DynamoDB table must have been previously created. If the DynamoDB table
was created within the last minute, please wait for a minute or two and try again.

Error: %s
`
