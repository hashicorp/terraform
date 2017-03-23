package s3

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	// This will be used as directory name, the odd looking colon is simply to
	// reduce the chance of name conflicts with existing objects.
	keyEnvPrefix = "env:"
)

func (b *Backend) States() ([]string, error) {
	params := &s3.ListObjectsInput{
		Bucket: &b.bucketName,
		Prefix: aws.String(keyEnvPrefix + "/"),
	}

	resp, err := b.s3Client.ListObjects(params)
	if err != nil {
		return nil, err
	}

	var envs []string
	for _, obj := range resp.Contents {
		env := keyEnv(*obj.Key)
		if env != "" {
			envs = append(envs, env)
		}
	}

	sort.Strings(envs)
	envs = append([]string{backend.DefaultStateName}, envs...)
	return envs, nil
}

// extract the env name from the S3 key
func keyEnv(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		// no env here
		return ""
	}

	if parts[0] != keyEnvPrefix {
		// not our key, so ignore
		return ""
	}

	return parts[1]
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	params := &s3.DeleteObjectInput{
		Bucket: &b.bucketName,
		Key:    aws.String(b.path(name)),
	}

	_, err := b.s3Client.DeleteObject(params)
	if err != nil {
		return err
	}

	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	client := &RemoteClient{
		s3Client:             b.s3Client,
		dynClient:            b.dynClient,
		bucketName:           b.bucketName,
		path:                 b.path(name),
		serverSideEncryption: b.serverSideEncryption,
		acl:                  b.acl,
		kmsKeyID:             b.kmsKeyID,
		lockTable:            b.lockTable,
	}

	// if this isn't the default state name, we need to create the object so
	// it's listed by States.
	if name != backend.DefaultStateName {
		if err := client.Put([]byte{}); err != nil {
			return nil, err
		}
	}

	return &remote.State{Client: client}, nil
}

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.keyName
	}

	return strings.Join([]string{keyEnvPrefix, name, b.keyName}, "/")
}
