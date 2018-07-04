package remote

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

type remoteClient struct {
	client       *tfe.Client
	organization string
	workspace    string
}

// Get the remote state.
func (r *remoteClient) Get() (*remote.Payload, error) {
	ctx := context.Background()

	// Retrieve the workspace for which to create a new state.
	w, err := r.client.Workspaces.Read(ctx, r.organization, r.workspace)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			// If no state exists, then return nil.
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving workspace: %v", err)
	}

	sv, err := r.client.StateVersions.Current(ctx, w.ID)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			// If no state exists, then return nil.
			return nil, nil
		}
		return nil, fmt.Errorf("Error retrieving remote state: %v", err)
	}

	state, err := r.client.StateVersions.Download(ctx, sv.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("Error downloading remote state: %v", err)
	}

	// If the state is empty, then return nil.
	if len(state) == 0 {
		return nil, nil
	}

	// Get the MD5 checksum of the state.
	sum := md5.Sum(state)

	return &remote.Payload{
		Data: state,
		MD5:  sum[:],
	}, nil
}

// Put the remote state.
func (r *remoteClient) Put(state []byte) error {
	ctx := context.Background()

	// Retrieve the workspace for which to create a new state.
	w, err := r.client.Workspaces.Read(ctx, r.organization, r.workspace)
	if err != nil {
		return fmt.Errorf("Error retrieving workspace: %v", err)
	}

	//  the state into a buffer.
	tfState, err := terraform.ReadState(bytes.NewReader(state))
	if err != nil {
		return fmt.Errorf("Error reading state: %s", err)
	}

	options := tfe.StateVersionCreateOptions{
		Lineage: tfe.String(tfState.Lineage),
		Serial:  tfe.Int64(tfState.Serial),
		MD5:     tfe.String(fmt.Sprintf("%x", md5.Sum(state))),
		State:   tfe.String(base64.StdEncoding.EncodeToString(state)),
	}

	// Create the new state.
	_, err = r.client.StateVersions.Create(ctx, w.ID, options)
	if err != nil {
		return fmt.Errorf("Error creating remote state: %v", err)
	}

	return nil
}

// Delete the remote state.
func (r *remoteClient) Delete() error {
	err := r.client.Workspaces.Delete(context.Background(), r.organization, r.workspace)
	if err != nil && err != tfe.ErrResourceNotFound {
		return fmt.Errorf("Error deleting workspace %s: %v", r.workspace, err)
	}

	return nil
}
