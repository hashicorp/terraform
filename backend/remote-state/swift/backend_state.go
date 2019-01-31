package swift

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

func getContainerAndPrefix(container string) (string, string) {
	var prefix string
	parts := strings.SplitN(container, "/", 2)
	if len(parts) > 1 {
		prefix = parts[1] + "/"
	}
	return parts[0], prefix
}

func (b *Backend) States() ([]string, error) {
	container, prefix := getContainerAndPrefix(b.container)

	listOpts := &objects.ListOpts{
		Prefix:    prefix,
		Delimiter: "/",
		Full:      false,
	}

	wss := []string{backend.DefaultStateName}

	allPages, err := objects.List(b.client, container, listOpts).AllPages()
	if err != nil {
		if _, ok := err.(gophercloud.ErrDefault404); ok {
			return wss, nil
		}
		return nil, err
	}

	objectList, err := objects.ExtractNames(allPages)
	if err != nil {
		return nil, fmt.Errorf("Unable to extract objects: %v", err)
	}

	for _, obj := range objectList {
		ws := b.validateWorkSpace(obj, prefix)
		if ws != "" {
			wss = append(wss, ws)
		}
	}

	sort.Strings(wss[1:])
	return wss, nil
}

func (b *Backend) validateWorkSpace(name, prefix string) string {
	name = strings.TrimPrefix(name, prefix)
	if name == DEFAULT_NAME+TFSTATE_SUFFIX {
		return ""
	}
	if strings.HasSuffix(name, TFSTATE_SUFFIX) {
		return strings.TrimSuffix(name, TFSTATE_SUFFIX)
	}
	return ""
}

func (b *Backend) DeleteState(name string) error {
	if name == DEFAULT_NAME || name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, fmt.Errorf("missing state name")
	}
	if name == DEFAULT_NAME {
		return nil, fmt.Errorf("invalid state name %s", name)
	}
	if name == backend.DefaultStateName {
		name = DEFAULT_NAME
	}

	container, prefix := getContainerAndPrefix(b.container)
	archiveContainer, _ := getContainerAndPrefix(b.archiveContainer)

	client := &RemoteClient{
		name:             name,
		client:           b.client,
		container:        container,
		prefix:           prefix,
		archive:          b.archive,
		archiveContainer: archiveContainer,
		expireSecs:       b.expireSecs,
	}

	return client, nil
}

func (b *Backend) State(name string) (state.State, error) {
	client, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	return &remote.State{Client: client}, nil
}
