package dynamodb

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
//	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	multierror "github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"

    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Store the last saved serial in dynamo with this suffix for consistency checks.
const (
	s3EncryptionAlgorithm  = "AES256"
	stateIDSuffix          = "-md5"
	s3ErrCodeInternalError = "InternalError"
	dynamoDBItemSize = 409600
)

type RemoteClient struct {
	s3Client              *s3.S3
	dynClient             *dynamodb.DynamoDB
	bucketName            string
	path                  string
	serverSideEncryption  bool
	customerEncryptionKey []byte
	acl                   string
	kmsKeyID              string
	ddbTable              string
}

type State struct {
    StateID string
    SegmentID string
    Body string
}

var (
	// The amount of time we will retry a state waiting for it to match the
	// expected checksum.
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
	    segmentID, err := strconv.Atoi(state.SegmentID)
		if err != nil {
	        fmt.Println("Got error unmarshalling:") // TO REMOVE
	        fmt.Println(err.Error()) // TO REMOVE
	        return 0, err
	    }
	    if segmentID > maxSegmentID{
	    	maxSegmentID = segmentID
	    }
	}
	return maxSegmentID, nil
}

func (c *RemoteClient) get() (*remote.Payload, error) {
	var output *s3.GetObjectOutput
	var err error

	input := &s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    &c.path,
	}

	if c.serverSideEncryption && c.customerEncryptionKey != nil {
		input.SetSSECustomerKey(string(c.customerEncryptionKey))
		input.SetSSECustomerAlgorithm(s3EncryptionAlgorithm)
		input.SetSSECustomerKeyMD5(c.getSSECustomerKeyMD5())
	}

	output, err = c.s3Client.GetObject(input)

	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			switch awserr.Code() {
			case s3.ErrCodeNoSuchBucket:
				return nil, fmt.Errorf(errS3NoSuchBucket, err)
			case s3.ErrCodeNoSuchKey:
				return nil, nil
			}
		}
		return nil, err
	}

	defer output.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}

	sum := md5.Sum(buf.Bytes())
	payload := &remote.Payload{
		Data: buf.Bytes(),
		MD5:  sum[:],
	}

/** DynamoDB **/
    tableName := "terraform-global-table-sort"

	var queryInput = &dynamodb.QueryInput{
	    TableName: aws.String(tableName),
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
	    return nil, err
	}

	maxSegmentID, err := getMaxSegmentId(result.Items)

	fmt.Println("maxSegmentID in get function: %d", maxSegmentID)

	var segmentStrings = make([]string, maxSegmentID+1)


	for _, i := range result.Items {
	    state := State{}

	    err = dynamodbattribute.UnmarshalMap(i, &state)

	   	segmentID, _ := strconv.Atoi(state.SegmentID)

	    segmentStrings[segmentID] = state.Body
	}

	jsonString := strings.Join(segmentStrings[:], "")

	fmt.Println("segmentStrings in get function: %d", jsonString)









/** DynamoDB **/

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *RemoteClient) GeberatePutItems(data []byte, sequence []int, transactionItems *[]*dynamodb.TransactWriteItem) error {
	body := string(data[:])

	item := State{
	    StateID: c.path,
	    SegmentID: strconv.Itoa(sequence[0]),
	    Body: body,
	}

	b, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("Got error marshalling item: %s", err)
	}

	if len(b) < dynamoDBItemSize {

		tableName := "terraform-global-table-sort" // TO CHANGE c.bucketName

		av, err := dynamodbattribute.MarshalMap(item)
		if err != nil {
		    return fmt.Errorf("Got error marshalling state: %s", err)
		}

		put_item := &dynamodb.TransactWriteItem{
			Put: &dynamodb.Put{
				TableName: aws.String(tableName),
				Item:      av,
			},
		}

		*transactionItems = append(*transactionItems, put_item)

	}else {
		N := int(len(data)/2)
		err := c.GeberatePutItems(data[N:], sequence[N:], transactionItems)
		if err != nil {
			return err
		}
		err = c.GeberatePutItems(data[:N], sequence[:N], transactionItems)
		if err != nil {
			return err
		}		
	}

	return nil
}

func GenerateSequence(seuqneceSize int, currentSegments []int) []int{
	segmentsSize := len(currentSegments)
	sequence := make([]int, seuqneceSize)
	position := 0
	for index := 0; index < seuqneceSize+segmentsSize; index++ {
		to_use := true
		for _,segment := range currentSegments{
			to_use = !(segment==index) && to_use
		}
		if to_use {
			sequence[position] = index
			position += 1
		}
	}
	return sequence
}

func (c *RemoteClient) Put(data []byte) error {
	contentType := "application/json"
	contentLength := int64(len(data))

	i := &s3.PutObjectInput{
		ContentType:   &contentType,
		ContentLength: &contentLength,
		Body:          bytes.NewReader(data),
		Bucket:        &c.bucketName,
		Key:           &c.path,
	}

	if c.serverSideEncryption {
		if c.kmsKeyID != "" {
			i.SSEKMSKeyId = &c.kmsKeyID
			i.ServerSideEncryption = aws.String("aws:kms")
		} else if c.customerEncryptionKey != nil {
			i.SetSSECustomerKey(string(c.customerEncryptionKey))
			i.SetSSECustomerAlgorithm(s3EncryptionAlgorithm)
			i.SetSSECustomerKeyMD5(c.getSSECustomerKeyMD5())
		} else {
			i.ServerSideEncryption = aws.String(s3EncryptionAlgorithm)
		}
	}

	if c.acl != "" {
		i.ACL = aws.String(c.acl)
	}

	log.Printf("[DEBUG] Uploading remote state to S3: %#v", i)

	_, err := c.s3Client.PutObject(i)
	if err != nil {
		return fmt.Errorf("failed to upload state: %s", err)
	}

	sum := md5.Sum(data)
	if err := c.putMD5(sum[:]); err != nil {
		// if this errors out, we unfortunately have to error out altogether,
		// since the next Get will inevitably fail.
		return fmt.Errorf("failed to store state MD5: %s", err)

	}

/** Dynamo DB **/
    tableName := "terraform-global-table-sort"

	var queryInput = &dynamodb.QueryInput{
	    TableName: aws.String(tableName),
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
	var segments []int
	for _, i := range result.Items {
	    state := State{}

	    err = dynamodbattribute.UnmarshalMap(i, &state)

	    if err != nil {
	        fmt.Println("Got error unmarshalling:") // TO REMOVE
	        fmt.Println(err.Error()) // TO REMOVE
	        return err
	    }
	    delete_item := &dynamodb.TransactWriteItem{
			Delete: &dynamodb.Delete{
				TableName: aws.String(tableName),
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
	        fmt.Println("Got error unmarshalling:") // TO REMOVE
	        fmt.Println(err.Error()) // TO REMOVE
	        return err
	    }
		segments = append(segments, id)
	    fmt.Println("StateID: ", state.StateID)
	}

	sequence := GenerateSequence(len(data), segments)
	err = c.GeberatePutItems(data, sequence, &transactionItems)
	if err != nil {
		return fmt.Errorf("Got error calling GeberatePutItems: %s", err)
	}

	_, err = c.dynClient.TransactWriteItems(&dynamodb.TransactWriteItemsInput{TransactItems: transactionItems})
	if err != nil {
		return fmt.Errorf("Got error calling TransactWriteItems: %s", err)
	}

/** Dynamo DB **/

	return nil
}

func (c *RemoteClient) Delete() error {
	_, err := c.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &c.bucketName,
		Key:    &c.path,
	})

	if err != nil {
		return err
	}

	if err := c.deleteMD5(); err != nil {
		log.Printf("error deleting state md5: %s", err)
	}

	return nil
}

func (c *RemoteClient) Lock(info *state.LockInfo) (string, error) {
	if c.ddbTable == "" {
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
		TableName:           aws.String(c.ddbTable),
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
	if c.ddbTable == "" {
		return nil, nil
	}

	getParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath() + stateIDSuffix)},
		},
		ProjectionExpression: aws.String("LockID, Digest"),
		TableName:            aws.String(c.ddbTable),
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
	if c.ddbTable == "" {
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
		TableName: aws.String(c.ddbTable),
	}
	_, err := c.dynClient.PutItem(putParams)
	if err != nil {
		log.Printf("[WARN] failed to record state serial in dynamodb: %s", err)
	}

	return nil
}

// remove the hash value for a deleted state
func (c *RemoteClient) deleteMD5() error {
	if c.ddbTable == "" {
		return nil
	}

	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(c.lockPath() + stateIDSuffix)},
		},
		TableName: aws.String(c.ddbTable),
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
		TableName:            aws.String(c.ddbTable),
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
	if c.ddbTable == "" {
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
		TableName: aws.String(c.ddbTable),
	}
	_, err = c.dynClient.DeleteItem(params)

	if err != nil {
		lockErr.Err = err
		return lockErr
	}
	return nil
}

func (c *RemoteClient) lockPath() string {
	return fmt.Sprintf("%s/%s", c.bucketName, c.path)
}

func (c *RemoteClient) getSSECustomerKeyMD5() string {
	b := md5.Sum(c.customerEncryptionKey)
	return base64.StdEncoding.EncodeToString(b[:])
}

const errBadChecksumFmt = `state data in S3 does not have the expected content.

This may be caused by unusually long delays in S3 processing a previous state
update.  Please wait for a minute or two and try again. If this problem
persists, and neither S3 nor DynamoDB are experiencing an outage, you may need
to manually verify the remote state and update the Digest value stored in the
DynamoDB table to the following value: %x
`

const errS3NoSuchBucket = `S3 bucket does not exist.

The referenced S3 bucket must have been previously created. If the S3 bucket
was created within the last minute, please wait for a minute or two and try
again.

Error: %s
`
