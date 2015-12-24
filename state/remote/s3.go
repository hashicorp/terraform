package remote

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/go-cleanhttp"
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

	accessKeyId := conf["access_key"]
	secretAccessKey := conf["secret_key"]

	credentialsProvider := credentials.NewChainCredentials([]credentials.Provider{
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     accessKeyId,
			SecretAccessKey: secretAccessKey,
			SessionToken:    "",
		}},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
		&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(session.New())},
	})

	// Make sure we got some sort of working credentials.
	_, err := credentialsProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("Unable to determine AWS credentials. Set the AWS_ACCESS_KEY_ID and "+
			"AWS_SECRET_ACCESS_KEY environment variables.\n(error was: %s)", err)
	}

	awsConfig := &aws.Config{
		Credentials: credentialsProvider,
		Region:      aws.String(regionName),
		HTTPClient:  cleanhttp.DefaultClient(),
	}
	sess := session.New(awsConfig)
	nativeClient := s3.New(sess)

	return &S3Client{
		nativeClient:         nativeClient,
		bucketName:           bucketName,
		keyName:              keyName,
		serverSideEncryption: serverSideEncryption,
		acl:                  acl,
	}, nil
}

type S3Client struct {
	nativeClient         *s3.S3
	bucketName           string
	keyName              string
	serverSideEncryption bool
	acl                  string
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
		i.ServerSideEncryption = aws.String("AES256")
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
