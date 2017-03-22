package s3

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	// This will be used a directory name, the odd looking colon is to reduce
	// the chance of name conflicts with existing deployments.
	keyEnvPrefix = "env:"
)

func (b *Backend) States() ([]string, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) DeleteState(name string) error {
	return backend.ErrNamedStatesNotSupported
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	//params := &s3.ListObjectsInput{
	//    Bucket:       &b.client.bucketName,
	//    Delimiter:    aws.String("Delimiter"),
	//    EncodingType: aws.String("EncodingType"),
	//    Marker:       aws.String("Marker"),
	//    MaxKeys:      aws.Int64(1),
	//    Prefix:       aws.String("env"),
	//    RequestPayer: aws.String("RequestPayer"),
	//}
	return nil
}

func (b *Backend) State(name string) (state.State, error) {
	if name != backend.DefaultStateName {
		return nil, backend.ErrNamedStatesNotSupported
	}

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

	// TODO: create new state if it doesn't exist

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
