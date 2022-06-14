package cloud

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

type remoteClient struct {
	client       *tfe.Client
	lockInfo     *statemgr.LockInfo
	organization string
	runID        string
	stateUpload  stateUpload
	workspace    *tfe.Workspace
	forcePush    bool
}

type stateUpload struct {
	ctxOpts    *terraform.ContextOpts
	services   *disco.Disco
	hasErrored bool
}

// Get the remote state.
func (r *remoteClient) Get() (*remote.Payload, error) {
	ctx := context.Background()

	sv, err := r.client.StateVersions.ReadCurrent(ctx, r.workspace.ID)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			// If no state exists, then return nil.
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving state: %v", err)
	}

	state, err := r.client.StateVersions.Download(ctx, sv.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("error downloading state: %v", err)
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

	// Read the raw state into a Terraform state.
	stateFile, err := statefile.Read(bytes.NewReader(state))
	if err != nil {
		return fmt.Errorf("error reading state: %s", err)
	}

	schemas, err := getSchemas(r.stateUpload.ctxOpts, r.stateUpload.services, stateFile.State)

	if err != nil {
		r.stateUpload.hasErrored = true
		return fmt.Errorf("error uploading state: %v", err)
	}
	jsonState, err := jsonstate.Marshal(stateFile, schemas)

	if err != nil {
		r.stateUpload.hasErrored = true
		return fmt.Errorf("error uploading state: %v", err)
	}

	options := tfe.StateVersionCreateOptions{
		Lineage:  tfe.String(stateFile.Lineage),
		Serial:   tfe.Int64(int64(stateFile.Serial)),
		MD5:      tfe.String(fmt.Sprintf("%x", md5.Sum(state))),
		State:    tfe.String(base64.StdEncoding.EncodeToString(state)),
		Force:    tfe.Bool(r.forcePush),
		ExtState: jsonState,
	}

	// If we have a run ID, make sure to add it to the options
	// so the state will be properly associated with the run.
	if r.runID != "" {
		options.Run = &tfe.Run{ID: r.runID}
	}

	// Create the new state.
	_, err = r.client.StateVersions.Create(ctx, r.workspace.ID, options)

	if err != nil {
		r.stateUpload.hasErrored = true
		return fmt.Errorf("error uploading state: %v", err)
	}

	return nil
}

func getSchemas(ctxOpts *terraform.ContextOpts, services *disco.Disco, state *states.State) (*terraform.Schemas, error) {
	var schemas *terraform.Schemas // to get our schemas we need a *terraform.Context, a *configs.Config and *states.State

	if ctxOpts == nil {
		panic("An unexpected error occurred when uploading state to Terraform Cloud") // This should be unlikely to happen because Cloud.ContextOps gets assign the value of opts.ContextOpts early in the process when preparing backend and running CLIInit()
	}

	// Get our context
	tfCtx, ctxDiags := terraform.NewContext(ctxOpts)
	if ctxDiags.HasErrors() {
		return schemas, fmt.Errorf("error uploading state to Terraform Cloud: %w", ctxDiags.Err())
	}

	// Get our config
	configDir, err := os.Getwd()
	if err != nil {
		return schemas, fmt.Errorf("error getting current directory: %w", err)
	}

	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: configDir + ".terraform/modules",
		Services:   services,
	})
	if err != nil {
		return schemas, fmt.Errorf("error uploading state to Terraform Cloud: %w", err)
	}

	config, configDiags := loader.LoadConfig(configDir)
	if configDiags.HasErrors() {
		return schemas, fmt.Errorf("error uploading state to Terraform Cloud: %w", errors.New(configDiags.Error()))
	}

	schemas, schemaDiags := tfCtx.Schemas(config, state)
	if schemaDiags.HasErrors() {
		return schemas, fmt.Errorf("error uploading state to Terraform Cloud: %w", schemaDiags.Err())
	}
	return schemas, nil
}

// Delete the remote state.
func (r *remoteClient) Delete() error {
	err := r.client.Workspaces.Delete(context.Background(), r.organization, r.workspace.Name)
	if err != nil && err != tfe.ErrResourceNotFound {
		return fmt.Errorf("error deleting workspace %s: %v", r.workspace.Name, err)
	}

	return nil
}

// EnableForcePush to allow the remote client to overwrite state
// by implementing remote.ClientForcePusher
func (r *remoteClient) EnableForcePush() {
	r.forcePush = true
}

// Lock the remote state.
func (r *remoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	ctx := context.Background()

	lockErr := &statemgr.LockError{Info: r.lockInfo}

	// Lock the workspace.
	_, err := r.client.Workspaces.Lock(ctx, r.workspace.ID, tfe.WorkspaceLockOptions{
		Reason: tfe.String("Locked by Terraform"),
	})
	if err != nil {
		if err == tfe.ErrWorkspaceLocked {
			lockErr.Info = info
			err = fmt.Errorf("%s (lock ID: \"%s/%s\")", err, r.organization, r.workspace.Name)
		}
		lockErr.Err = err
		return "", lockErr
	}

	r.lockInfo = info

	return r.lockInfo.ID, nil
}

// Unlock the remote state.
func (r *remoteClient) Unlock(id string) error {
	ctx := context.Background()

	// We first check if there was an error while uploading the latest
	// state. If so, we will not unlock the workspace to prevent any
	// changes from being applied until the correct state is uploaded.
	if r.stateUpload.hasErrored {
		return nil
	}

	lockErr := &statemgr.LockError{Info: r.lockInfo}

	// With lock info this should be treated as a normal unlock.
	if r.lockInfo != nil {
		// Verify the expected lock ID.
		if r.lockInfo.ID != id {
			lockErr.Err = fmt.Errorf("lock ID does not match existing lock")
			return lockErr
		}

		// Unlock the workspace.
		_, err := r.client.Workspaces.Unlock(ctx, r.workspace.ID)
		if err != nil {
			lockErr.Err = err
			return lockErr
		}

		return nil
	}

	// Verify the optional force-unlock lock ID.
	if r.organization+"/"+r.workspace.Name != id {
		lockErr.Err = fmt.Errorf(
			"lock ID %q does not match existing lock ID \"%s/%s\"",
			id,
			r.organization,
			r.workspace.Name,
		)
		return lockErr
	}

	// Force unlock the workspace.
	_, err := r.client.Workspaces.ForceUnlock(ctx, r.workspace.ID)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	return nil
}
