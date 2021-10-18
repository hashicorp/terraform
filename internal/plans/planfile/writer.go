package planfile

import (
	"archive/zip"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

type CreateArgs struct {
	// ConfigSnapshot is a snapshot of the configuration that the plan
	// was created from.
	ConfigSnapshot *configload.Snapshot

	// PreviousRunStateFile is a representation of the state snapshot we used
	// as the original input when creating this plan, containing the same
	// information as recorded at the end of the previous apply except for
	// upgrading managed resource instance data to the provider's latest
	// schema versions.
	PreviousRunStateFile *statefile.File

	// BaseStateFile is a representation of the state snapshot we used to
	// create the plan, which is the result of asking the providers to refresh
	// all previously-stored objects to match the current situation in the
	// remote system. (If this plan was created with refreshing disabled,
	// this should be the same as PreviousRunStateFile.)
	StateFile *statefile.File

	// Plan records the plan itself, which is the main artifact inside a
	// saved plan file.
	Plan *plans.Plan

	// DependencyLocks records the dependency lock information that we
	// checked prior to creating the plan, so we can make sure that all of the
	// same dependencies are still available when applying the plan.
	DependencyLocks *depsfile.Locks
}

// Create creates a new plan file with the given filename, overwriting any
// file that might already exist there.
//
// A plan file contains both a snapshot of the configuration and of the latest
// state file in addition to the plan itself, so that Terraform can detect
// if the world has changed since the plan was created and thus refuse to
// apply it.
func Create(filename string, args CreateArgs) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// tfplan file
	{
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     tfplanFilename,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create tfplan file: %s", err)
		}
		err = writeTfplan(args.Plan, w)
		if err != nil {
			return fmt.Errorf("failed to write plan: %s", err)
		}
	}

	// tfstate file
	{
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     tfstateFilename,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create embedded tfstate file: %s", err)
		}
		err = statefile.Write(args.StateFile, w)
		if err != nil {
			return fmt.Errorf("failed to write state snapshot: %s", err)
		}
	}

	// tfstate-prev file
	{
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     tfstatePreviousFilename,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create embedded tfstate-prev file: %s", err)
		}
		err = statefile.Write(args.PreviousRunStateFile, w)
		if err != nil {
			return fmt.Errorf("failed to write previous state snapshot: %s", err)
		}
	}

	// tfconfig directory
	{
		err := writeConfigSnapshot(args.ConfigSnapshot, zw)
		if err != nil {
			return fmt.Errorf("failed to write config snapshot: %s", err)
		}
	}

	// .terraform.lock.hcl file, containing dependency lock information
	if args.DependencyLocks != nil { // (this was a later addition, so not all callers set it, but main callers should)
		src, diags := depsfile.SaveLocksToBytes(args.DependencyLocks)
		if diags.HasErrors() {
			return fmt.Errorf("failed to write embedded dependency lock file: %s", diags.Err().Error())
		}

		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     dependencyLocksFilename,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to create embedded dependency lock file: %s", err)
		}
		_, err = w.Write(src)
		if err != nil {
			return fmt.Errorf("failed to write embedded dependency lock file: %s", err)
		}
	}

	return nil
}
