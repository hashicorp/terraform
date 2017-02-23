package remote

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-multierror"
	uuid "github.com/hashicorp/go-uuid"
	terraformAws "github.com/hashicorp/terraform/builtin/providers/aws"
	"github.com/hashicorp/terraform/state"
)

func s3Factory(conf map[string]string) (Client, error) {
	bucketName, ok := conf["bucket"]
	if !ok {
		return nil, fmt.Errorf("missing 'bucket' configuration")
	}

	keyName, ok := conf["key"]
	if !ok {
		return nil, fmt.Errorf("missing 'key' configuration")
	}

	endpoint, ok := conf["endpoint"]
	if !ok {
		endpoint = os.Getenv("AWS_S3_ENDPOINT")
	}

	regionName, ok := conf["region"]
	if !ok {
		regionName = os.Getenv("AWS_DEFAULT_REGION")
		if regionName == "" {
			return nil, fmt.Errorf(
				"missing 'region' configuration or AWS_DEFAULT_REGION environment variable")
		}
	}

	serverSideEncryption := false
	if raw, ok := conf["encrypt"]; ok {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf(
				"'encrypt' field couldn't be parsed as bool: %s", err)
		}

		serverSideEncryption = v
	}

	acl := ""
	if raw, ok := conf["acl"]; ok {
		acl = raw
	}
	kmsKeyID := conf["kms_key_id"]

	var errs []error
	creds, err := terraformAws.GetCredentials(&terraformAws.Config{
		AccessKey:     conf["access_key"],
		SecretKey:     conf["secret_key"],
		Token:         conf["token"],
		Profile:       conf["profile"],
		CredsFilename: conf["shared_credentials_file"],
		AssumeRoleARN: conf["role_arn"],
	})
	if err != nil {
		return nil, err
	}

	// Call Get to check for credential provider. If nothing found, we'll get an
	// error, and we can present it nicely to the user
	_, err = creds.Get()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			errs = append(errs, fmt.Errorf(`No valid credential sources found for AWS S3 remote.
Please see https://www.terraform.io/docs/state/remote/s3.html for more information on
providing credentials for the AWS S3 remote`))
		} else {
			errs = append(errs, fmt.Errorf("Error loading credentials for AWS S3 remote: %s", err))
		}
		return nil, &multierror.Error{Errors: errs}
	}

	awsConfig := &aws.Config{
		Credentials: creds,
		Endpoint:    aws.String(endpoint),
		Region:      aws.String(regionName),
		HTTPClient:  cleanhttp.DefaultClient(),
	}
	sess := session.New(awsConfig)
	nativeClient := s3.New(sess)
	dynClient := dynamodb.New(sess)

	return &S3Client{
		nativeClient:         nativeClient,
		bucketName:           bucketName,
		keyName:              keyName,
		serverSideEncryption: serverSideEncryption,
		acl:                  acl,
		kmsKeyID:             kmsKeyID,
		dynClient:            dynClient,
		lockTable:            conf["lock_table"],
	}, nil
}

type S3Client struct {
	nativeClient         *s3.S3
	bucketName           string
	keyName              string
	serverSideEncryption bool
	acl                  string
	kmsKeyID             string
	dynClient            *dynamodb.DynamoDB
	lockTable            string
}

func (c *S3Client) Get() (*Payload, error) {
	output, err := c.nativeClient.GetObject(&s3.GetObjectInput{
		Bucket: &c.bucketName,
		Key:    &c.keyName,
	})

	if err != nil {
		if awserr := err.(awserr.Error); awserr != nil {
			if awserr.Code() == "NoSuchKey" {
				return nil, nil
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	defer output.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, output.Body); err != nil {
		return nil, fmt.Errorf("Failed to read remote state: %s", err)
	}

	payload := &Payload{
		Data: buf.Bytes(),
	}

	// If there was no data, then return nil
	if len(payload.Data) == 0 {
		return nil, nil
	}

	return payload, nil
}

func (c *S3Client) Put(data []byte) error {
	contentType := "application/json"
	contentLength := int64(len(data))

	i := &s3.PutObjectInput{
		ContentType:   &contentType,
		ContentLength: &contentLength,
		Body:          bytes.NewReader(data),
		Bucket:        &c.bucketName,
		Key:           &c.keyName,
	}

	if c.serverSideEncryption {
		if c.kmsKeyID != "" {
			i.SSEKMSKeyId = &c.kmsKeyID
			i.ServerSideEncryption = aws.String("aws:kms")
		} else {
			i.ServerSideEncryption = aws.String("AES256")
		}
	}

	if c.acl != "" {
		i.ACL = aws.String(c.acl)
	}

	log.Printf("[DEBUG] Uploading remote state to S3: %#v", i)

	if _, err := c.nativeClient.PutObject(i); err == nil {
		return nil
	} else {
		return fmt.Errorf("Failed to upload state: %v", err)
	}
}

func (c *S3Client) Delete() error {
	_, err := c.nativeClient.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &c.bucketName,
		Key:    &c.keyName,
	})

	return err
}

func (c *S3Client) Lock(info *state.LockInfo) (string, error) {
	if c.lockTable == "" {
		return "", nil
	}

	stateName := fmt.Sprintf("%s/%s", c.bucketName, c.keyName)
	info.Path = stateName

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}

		info.ID = lockID
	}

	putParams := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(stateName)},
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

func (c *S3Client) getLockInfo() (*state.LockInfo, error) {
	getParams := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"LockID": {S: aws.String(fmt.Sprintf("%s/%s", c.bucketName, c.keyName))},
		},
		ProjectionExpression: aws.String("LockID, Info"),
		TableName:            aws.String(c.lockTable),
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

func (c *S3Client) Unlock(id string) error {
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
			"LockID": {S: aws.String(fmt.Sprintf("%s/%s", c.bucketName, c.keyName))},
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
