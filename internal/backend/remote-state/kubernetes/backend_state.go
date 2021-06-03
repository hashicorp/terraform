package kubernetes

import (
	"errors"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Workspaces returns a list of names for the workspaces found in k8s. The default
// workspace is always returned as the first element in the slice.
func (b *Backend) Workspaces() ([]string, error) {
	secretClient, err := b.KubernetesSecretClient()
	if err != nil {
		return nil, err
	}

	secrets, err := secretClient.List(
		metav1.ListOptions{
			LabelSelector: tfstateKey + "=true",
		},
	)
	if err != nil {
		return nil, err
	}

	// Use a map so there aren't duplicate workspaces
	m := make(map[string]struct{})
	for _, secret := range secrets.Items {
		sl := secret.GetLabels()
		ws, ok := sl[tfstateWorkspaceKey]
		if !ok {
			continue
		}

		key, ok := sl[tfstateSecretSuffixKey]
		if !ok {
			continue
		}

		// Make sure it isn't default and the key matches
		if ws != backend.DefaultStateName && key == b.nameSuffix {
			m[ws] = struct{}{}
		}
	}

	states := []string{backend.DefaultStateName}
	for k := range m {
		states = append(states, k)
	}

	sort.Strings(states[1:])
	return states, nil
}

func (b *Backend) DeleteWorkspace(name string) error {
	if name == backend.DefaultStateName || name == "" {
		return fmt.Errorf("can't delete default state")
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return err
	}

	return client.Delete()
}

func (b *Backend) StateMgr(name string) (statemgr.Full, error) {
	c, err := b.remoteClient(name)
	if err != nil {
		return nil, err
	}

	stateMgr := &remote.State{Client: c}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		return nil, err
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {

		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, err
		}

		secretName, err := c.createSecretName()
		if err != nil {
			return nil, err
		}

		// Local helper function so we can call it multiple places
		unlock := func(baseErr error) error {
			if err := stateMgr.Unlock(lockID); err != nil {
				const unlockErrMsg = `%v
				Additionally, unlocking the state in Kubernetes failed:

				Error message: %q
				Lock ID (gen): %v
				Secret Name: %v

				You may have to force-unlock this state in order to use it again.
				The Kubernetes backend acquires a lock during initialization to ensure
				the initial state file is created.`
				return fmt.Errorf(unlockErrMsg, baseErr, err.Error(), lockID, secretName)
			}

			return baseErr
		}

		if err := stateMgr.WriteState(states.NewState()); err != nil {
			return nil, unlock(err)
		}
		if err := stateMgr.PersistState(); err != nil {
			return nil, unlock(err)
		}

		// Unlock, the state should now be initialized
		if err := unlock(nil); err != nil {
			return nil, err
		}

	}

	return stateMgr, nil
}

// get a remote client configured for this state
func (b *Backend) remoteClient(name string) (*RemoteClient, error) {
	if name == "" {
		return nil, errors.New("missing state name")
	}

	secretClient, err := b.KubernetesSecretClient()
	if err != nil {
		return nil, err
	}

	leaseClient, err := b.KubernetesLeaseClient()
	if err != nil {
		return nil, err
	}

	client := &RemoteClient{
		kubernetesSecretClient: secretClient,
		kubernetesLeaseClient:  leaseClient,
		namespace:              b.namespace,
		labels:                 b.labels,
		nameSuffix:             b.nameSuffix,
		workspace:              name,
	}

	return client, nil
}

func (b *Backend) client() *RemoteClient {
	return &RemoteClient{}
}
