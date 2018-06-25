package swift

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

const (
	objectEnvPrefix = "env-"
)

func (b *Backend) States() ([]string, error) {
	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
	}

	// List our container objects
	objectNames, err := client.ListObjectsNames(objectEnvPrefix)

	if err != nil {
		return nil, err
	}

	// Find the envs, we use a map since we can get duplicates with
	// path suffixes.
	envs := map[string]struct{}{}

	for _, object := range objectNames {
		object = strings.TrimPrefix(object, objectEnvPrefix)

		// Ignore anything with a "/" in it since we store the state
		// directly in a key not a directory.
		if idx := strings.IndexRune(object, '/'); idx >= 0 {
			continue
		}

		envs[object] = struct{}{}
	}

	if err != nil {
		return nil, err
	}

	result := make([]string, 1, len(envs)+1)
	result[0] = backend.DefaultStateName

	for k, _ := range envs {
		result = append(result, k)
	}

	return result, nil

}

func (b *Backend) DeleteState(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
		objectName:       b.objectName(name),
	}

	// List our container objects
	err := client.Delete()
	return err

}

func (b *Backend) State(name string) (state.State, error) {
	if name == "" {
		return nil, fmt.Errorf("missing state name")
	}

	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
		objectName:       b.objectName(name),
	}

	stateMgr := &remote.State{Client: client}

	//if this isn't the default state name, we need to create the object so
	//it's listed by States.
	if name != backend.DefaultStateName {
		// Grab the value
		if err := stateMgr.RefreshState(); err != nil {
			return nil, err
		}

		// If we have no state, we have to create an empty state
		if v := stateMgr.State(); v == nil {
			if err := stateMgr.WriteState(terraform.NewState()); err != nil {
				return nil, err
			}
			if err := stateMgr.PersistState(); err != nil {
				return nil, err
			}
		}
	}

	return stateMgr, nil
}

func (b *Backend) objectName(name string) string {
	if name != backend.DefaultStateName {
		name = fmt.Sprintf("%s%s", objectEnvPrefix, name)
	}
	return name
}
