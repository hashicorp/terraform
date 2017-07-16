package b2

import (
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"

	"errors"
	"gopkg.in/kothar/go-backblaze.v0"
	"sort"
	"strings"
)

const (
	keyEnvSuffix = "env:"
)

func (b *Backend) States() ([]string, error) {
	bucket, err := b.b2.Bucket(b.bucketName)
	if err != nil {
		return nil, err
	}

	filenames, err := b.findFilesWithEnvSuffix(keyEnvSuffix, bucket)
	if err != nil {
		return nil, err
	}

	envs := []string{backend.DefaultStateName}
	for _, filename := range filenames {
		parts := strings.SplitN(filename, keyEnvSuffix, 2)
		if parts[1] != "" {
			envs = append(envs, parts[1])
		}
	}

	sort.Strings(envs[1:])
	return envs, nil
}

// go-backblaze currently doesn't support the B2 API's 'prefix' option
// Until this is fixed, start from default env's path and manually check for env suffix
func (b *Backend) findFilesWithEnvSuffix(suffix string, bucket *backblaze.Bucket) ([]string, error) {
	scanning := true
	filenames := []string{}
	startingFilename := b.keyName

	for scanning {
		resp, err := bucket.ListFileNames(startingFilename, 1000)
		if err != nil {
			return nil, err
		}

		for _, fileStatus := range resp.Files {
			if strings.HasPrefix(fileStatus.Name, b.keyName) {
				filenames = append(filenames, fileStatus.Name)
			}
		}

		if strings.HasPrefix(resp.NextFileName, b.keyName) {
			startingFilename = resp.NextFileName
		} else {
			scanning = false
		}
	}

	return filenames, nil
}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return errors.New("can't delete default state")
	}

	bucket, err := b.b2.Bucket(b.bucketName)
	if err != nil {
		return err
	}

	_, err = bucket.HideFile(b.path(name))

	return err
}

func (b *Backend) State(name string) (state.State, error) {
	return nil, backend.ErrNamedStatesNotSupported
}

func (b *Backend) path(name string) string {
	if name == backend.DefaultStateName {
		return b.keyName
	}

	return b.keyName + keyEnvSuffix + name
}
