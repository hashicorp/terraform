// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Workspaces returns a list of names for the workspaces found in k8s. The default
// workspace is always returned as the first element in the slice.
func (b *Backend) Workspaces() ([]string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	secretClient, err := b.KubernetesSecretClient()
	if err != nil {
		return nil, diags.Append(err)
	}

	secrets, err := secretClient.List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: tfstateKey + "=true",
		},
	)
	if err != nil {
		return nil, diags.Append(err)
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
	return states, diags
}

func (b *Backend) DeleteWorkspace(name string, _ bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if name == backend.DefaultStateName || name == "" {
		return diags.Append(fmt.Errorf("can't delete default state"))
	}

	client, err := b.remoteClient(name)
	if err != nil {
		return diags.Append(err)
	}

	return diags.Append(client.Delete())
}

func (b *Backend) StateMgr(name string) (statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	c, err := b.remoteClient(name)
	if err != nil {
		return nil, diags.Append(err)
	}

	stateMgr := &remote.State{Client: c}

	// Grab the value
	if err := stateMgr.RefreshState(); err != nil {
		return nil, diags.Append(err)
	}

	// If we have no state, we have to create an empty state
	if v := stateMgr.State(); v == nil {

		lockInfo := statemgr.NewLockInfo()
		lockInfo.Operation = "init"
		lockID, err := stateMgr.Lock(lockInfo)
		if err != nil {
			return nil, diags.Append(err)
		}

		// get base secret name
		secretName, err := c.createSecretName(0)
		if err != nil {
			return nil, diags.Append(err)
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
			unlockErr := unlock(err)
			return nil, diags.Append(unlockErr)
		}
		if err := stateMgr.PersistState(nil); err != nil {
			unlockErr := unlock(err)
			return nil, diags.Append(unlockErr)
		}

		// Unlock, the state should now be initialized
		if err := unlock(nil); err != nil {
			return nil, diags.Append(err)
		}

	}

	return stateMgr, diags
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
