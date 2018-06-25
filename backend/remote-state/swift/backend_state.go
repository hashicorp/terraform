package swift

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/states"
)

const (
	objectEnvPrefix = "env-"
	delimiter       = "/"
)

func (b *Backend) Workspaces() ([]string, error) {
	client := &RemoteClient{
		client:           b.client,
		container:        b.container,
		archive:          b.archive,
		archiveContainer: b.archiveContainer,
		expireSecs:       b.expireSecs,
	}

	// List our container objects
	objectNames, err := client.ListObjectsNames(objectEnvPrefix, delimiter)

	if err != nil {
		return nil, err
	}

	// Find the envs, we use a map since we can get duplicates with
	// path suffixes.
	envs := map[string]struct{}{}

	for _, object := range objectNames {
		object = strings.TrimPrefix(object, objectEnvPrefix)
		object = strings.TrimSuffix(object, delimiter)

		// Ignore objects that still contain a "/"
		// as we dont store states in subdirectories
		if idx := strings.Index(object, delimiter); idx >= 0 {
			continue
		}

		envs[object] = struct{}{}
	}

	result := make([]string, 1, len(envs)+1)
	result[0] = backend.DefaultStateName

	for k, _ := range envs {
		result = append(result, k)
	}

	return result, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
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

func (b *Backend) StateMgr(name string) (state.State, error) {
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
			if err := stateMgr.WriteState(states.NewState()); err != nil {
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
		name = fmt.Sprintf("%s%s/%s", objectEnvPrefix, name, TFSTATE_NAME)
	} else {
		name = TFSTATE_NAME
	}

	return name
}
