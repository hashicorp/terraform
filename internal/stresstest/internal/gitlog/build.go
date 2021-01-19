package gitlog

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"

	"github.com/apparentlymart/go-mingit/mingit"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressrun"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

// NoParent is a special object ID used to represent the absense of a parent
// when calling BuildCommitForLogStep.
var NoParent mingit.ObjectID

// BuildCommitForLogStep is the main function in this package, encapsulating
// all of the work to represent a particular stressrun.LogStep as a git commit.
//
// Since git is content-addressible, it's inconvenient to construct a git
// commit without simultaneously writing it into a repository, and so this
// function also has the side-effect of writing various blob and tree objects,
// along with the single commit, into the given repository. However, it doesn't
// make any changes to the repository's refs, so the caller is free to decide
// how (and whether) to associate the generated commit with a branch.
//
// If you're creating the initial commit for a repository, set parentID to
// gitlog.NoParent to represent the absense of a parent. Run logs are linear,
// so (unlike generalized git commits) log step commits always have only
// zero or one parents.
func BuildCommitForLogStep(step stressrun.LogStep, repo *mingit.Repository, parentID mingit.ObjectID) (mingit.ObjectID, error) {
	// When building a git commit we need to work upwards from the leaves of
	// the tree (the files), through the nested directories, and then finally
	// the root commit.

	identity := mingit.Identity{
		Name:  "Terraform Stresstest",
		Email: "stresstest@invalid",
		Time:  step.Time,
	}
	tree, err := buildTreeForLogStep(step, repo)
	if err != nil {
		return NoParent, err
	}
	commit := &mingit.Commit{
		TreeID:    tree,
		Message:   step.Message,
		Author:    &identity,
		Committer: &identity,
	}
	if parentID.String() != NoParent.String() {
		commit.ParentIDs = []mingit.ObjectID{parentID}
	}
	return repo.WriteCommit(commit)
}

// BuildCommitsForLog is a handy wrapper around BuildCommitForLogStep that
// iterates over all of the steps in the given log and constructs a series
// of commits from them, using BuildCommitForLogStep on each one.
//
// A successful result is a pair of commit object ids, with the first
// representing the last step that was completed and the second representing
// the actual last step in the log. These can differ in the case where an
// intermediate step failed and thus blocked the steps that followed. In that
// case, the caller might want to create two different refs in the repository,
// so the person using the repository can then try to apply the remaining
// steps once they've fixed a bug, if that's helpful to their debugging.
func BuildCommitsForLog(log stressrun.Log, repo *mingit.Repository) (latest, all mingit.ObjectID, err error) {
	previousID := NoParent
	latestID := NoParent // the commit for the final non-blocked step
	for _, step := range log {
		id, err := BuildCommitForLogStep(step, repo, previousID)
		if err != nil {
			return NoParent, NoParent, err
		}
		previousID = id
		if step.Status != stressrun.StepBlocked {
			latestID = id
		}
	}
	return latestID, previousID, nil
}

func buildTreeForLogStep(step stressrun.LogStep, repo *mingit.Repository) (mingit.ObjectID, error) {
	snap := step.Config.ConfigSnapshot()
	tree, err := treeForConfigSnapshot(snap, repo)
	if err != nil {
		return NoParent, err
	}
	// For the _root_ tree in a step, we also have some other artifacts to
	// append: the state snapshot, the provider fake data snapshot, and
	// the terraform.tfvars file.

	stateBlobID, err := buildBlobForStateSnapshot(step.StateSnapshot, repo)
	if err != nil {
		return NoParent, err
	}
	tree = append(tree, mingit.TreeItem{
		Mode:     mingit.ModeRegular,
		Name:     "terraform.tfstate",
		TargetID: stateBlobID,
	})

	tfvarsBlobID, err := buildBlobForInputVariables(step.Config.VariableValues(), repo)
	if err != nil {
		return NoParent, err
	}
	tree = append(tree, mingit.TreeItem{
		Mode:     mingit.ModeRegular,
		Name:     "terraform.tfvars",
		TargetID: tfvarsBlobID,
	})

	remoteObjsTreeID, err := buildTreeForRemoteObjects(step.RemoteObjects, repo)
	if err != nil {
		return NoParent, err
	}
	tree = append(tree, mingit.TreeItem{
		Mode:     mingit.ModeDir,
		Name:     "remote-objects",
		TargetID: remoteObjsTreeID,
	})

	gitignoreBlobID, err := repo.WriteBlob(gitignore)
	if err != nil {
		return NoParent, err
	}
	tree = append(tree, mingit.TreeItem{
		Mode:     mingit.ModeRegular,
		Name:     ".gitignore",
		TargetID: gitignoreBlobID,
	})

	return repo.WriteTree(tree)
}

func treeForConfigSnapshot(snap *configload.Snapshot, repo *mingit.Repository) (mingit.Tree, error) {
	// This is a pretty awkward operation because we need to produce a nested
	// tree structure reflecting the module heirarchy, but the
	// configload.Snapshot format is just a flat set of modules. To make this
	// work without too much complexity, we'll repeatedly scan over the modules
	// in the snapshot and do parsing on their paths in order to recover the
	// effective tree structure. We're assuming that snapshots only have tens
	// of modules and so this won't be a big deal.
	// This approach is not generic for all snapshots, but it works here
	// because the stressgen package happens to generate a filesystem layout
	// which exactly matches the module tree.
	return treeForConfigSnapshotDir(snap, ".", repo)
}

func treeForConfigSnapshotDir(snap *configload.Snapshot, subpath string, repo *mingit.Repository) (mingit.Tree, error) {
	var tree mingit.Tree

	subpath = path.Clean(subpath)

	// Our goal here is to search the module entries in the snapshot to find
	// the ones whose paths have subpath as a prefix. We'll recursively
	// visit each of these to walk the whole directory tree.
	// We also need to find a module whose path _is_ subpath, and append
	// all of the files inside it to get the source files for the current
	// module.
	for _, mod := range snap.Modules {
		if path.Clean(mod.Dir) == subpath {
			// We've found the object representing the current directory, so
			// we'll append the files from it.
			for fn, content := range mod.Files {
				blobID, err := repo.WriteBlob(content)
				if err != nil {
					return tree, fmt.Errorf("failed to create blob for %s: %w", path.Join(subpath, fn), err)
				}
				tree = append(tree, mingit.TreeItem{
					Mode:     mingit.ModeRegular,
					Name:     fn,
					TargetID: blobID,
				})
			}
			continue
		}

		parent, dirName := path.Split(mod.Dir)
		parent = filepath.Clean(parent)
		if parent != "" && parent == subpath {
			subsubpath := path.Join(subpath, dirName)
			subTree, err := treeForConfigSnapshotDir(snap, subsubpath, repo)
			if err != nil {
				return tree, fmt.Errorf("failed to create tree for %s: %w", subsubpath, err)
			}
			treeID, err := repo.WriteTree(subTree)
			if err != nil {
				return tree, fmt.Errorf("failed to create tree for %s: %w", subsubpath, err)
			}
			tree = append(tree, mingit.TreeItem{
				Mode:     mingit.ModeDir,
				Name:     dirName,
				TargetID: treeID,
			})
		}
	}

	return tree, nil
}

func buildBlobForStateSnapshot(state *states.State, repo *mingit.Repository) (mingit.ObjectID, error) {
	file := statefile.New(state, "stresstest", 1)
	var buf bytes.Buffer
	err := statefile.Write(file, &buf)
	if err != nil {
		return NoParent, fmt.Errorf("failed to serialize state: %w", err)
	}
	return repo.WriteBlob(buf.Bytes())
}

func buildBlobForInputVariables(vals map[string]cty.Value, repo *mingit.Repository) (mingit.ObjectID, error) {
	// Using SerializeResourceObject here is a bit of a cheat, because this
	// isn't actually an instance of a resource, but that constraint is mainly
	// for external callers to this package and we happen to know that
	// the current implementation is also good enough for our variables,
	// because stresstest happens to always set all variables to string values.
	src := SerializeResourceObject(cty.ObjectVal(vals))
	return repo.WriteBlob(src)
}

func buildTreeForRemoteObjects(objs map[string]cty.Value, repo *mingit.Repository) (mingit.ObjectID, error) {
	tree := make(mingit.Tree, 0, len(objs))
	for id, obj := range objs {
		// We serialize each of the objects in a "tfvars-like" file because
		// that makes it easier to compare with the input and generally
		// diffs better than JSON does.
		src := SerializeResourceObject(obj)
		blobID, err := repo.WriteBlob(src)
		if err != nil {
			return NoParent, fmt.Errorf("failed to serialize %s: %w", id, err)
		}
		tree = append(tree, mingit.TreeItem{
			Mode:     mingit.ModeRegular,
			Name:     id,
			TargetID: blobID,
		})
	}
	return repo.WriteTree(tree)
}

// SerializeResourceObject returns a serialized version of the given object
// (assumed to be an instance of the stressful managed resource type) which
// can therefore be included as a file in a git-based log.
//
// This uses a .tfvars-like HCL syntax to make it look similar to the input
// and because it typically diffs better than JSON does.
func SerializeResourceObject(obj cty.Value) []byte {
	f := hclwrite.NewEmptyFile()
	body := f.Body()
	for it := obj.ElementIterator(); it.Next(); {
		name, val := it.Element()
		body.SetAttributeValue(name.AsString(), val)
	}
	return f.Bytes()
}

var gitignore = []byte(`.terraform/*
`)
