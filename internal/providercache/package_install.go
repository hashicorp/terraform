package providercache

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	getter "github.com/hashicorp/go-getter"

	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/internal/copydir"
)

// We borrow the "unpack a zip file into a target directory" logic from
// go-getter, even though we're not otherwise using go-getter here.
// (We don't need the same flexibility as we have for modules, because
// providers _always_ come from provider registries, which have a very
// specific protocol and set of expectations.)
var unzip = getter.ZipDecompressor{}

func installFromHTTPURL(ctx context.Context, url string, targetDir string) error {
	// When we're installing from an HTTP URL we expect the URL to refer to
	// a zip file. We'll fetch that into a temporary file here and then
	// delegate to installFromLocalArchive below to actually extract it.
	// (We're not using go-getter here because its HTTP getter has a bunch
	// of extraneous functionality we don't need or want, like indirection
	// through X-Terraform-Get header, attempting partial fetches for
	// files that already exist, etc.)

	httpClient := httpclient.New()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("invalid provider download request: %s", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unsuccessful request to %s: %s", url, resp.Status)
	}

	f, err := ioutil.TempFile("", "terraform-provider")
	if err != nil {
		return fmt.Errorf("failed to open temporary file to download from %s", url)
	}
	defer f.Close()

	// We'll borrow go-getter's "cancelable copy" implementation here so that
	// the download can potentially be interrupted partway through.
	n, err := getter.Copy(ctx, f, resp.Body)
	if err == nil && n < resp.ContentLength {
		err = fmt.Errorf("incorrect response size: expected %d bytes, but got %d bytes", resp.ContentLength, n)
	}
	if err != nil {
		return err
	}

	// If we managed to download successfully then we can now delegate to
	// installFromLocalArchive for extraction.
	archiveFilename := f.Name()
	return installFromLocalArchive(ctx, archiveFilename, targetDir)
}

func installFromLocalArchive(ctx context.Context, filename string, targetDir string) error {
	return unzip.Decompress(targetDir, filename, true)
}

// installFromLocalDir is the implementation of both installing a package from
// a local directory source _and_ of linking a package from another cache
// in LinkFromOtherCache, because they both do fundamentally the same
// operation: symlink if possible, or deep-copy otherwise.
func installFromLocalDir(ctx context.Context, sourceDir string, targetDir string) error {
	absNew, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to make target path %s absolute: %s", targetDir, err)
	}
	absCurrent, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to make source path %s absolute: %s", sourceDir, err)
	}

	// Before we do anything else, we'll do a quick check to make sure that
	// these two paths are not pointing at the same physical directory on
	// disk. This compares the files by their OS-level device and directory
	// entry identifiers, not by their virtual filesystem paths.
	if same, err := copydir.SameFile(absNew, absCurrent); same {
		return fmt.Errorf("cannot install existing provider directory %s to itself", targetDir)
	} else if err != nil {
		return fmt.Errorf("failed to determine if %s and %s are the same: %s", sourceDir, targetDir, err)
	}

	// Delete anything that's already present at this path first.
	err = os.RemoveAll(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing %s before linking it to %s: %s", sourceDir, targetDir, err)
	}

	// We'll prefer to create a symlink if possible, but we'll fall back to
	// a recursive copy if symlink creation fails. It could fail for a number
	// of reasons, including being on Windows 8 without administrator
	// privileges or being on a legacy filesystem like FAT that has no way
	// to represent a symlink. (Generalized symlink support for Windows was
	// introduced in a Windows 10 minor update.)
	//
	// We use an absolute path for the symlink to reduce the risk of it being
	// broken by moving things around later, since the source directory is
	// likely to be a shared directory independent on any particular target
	// and thus we can't assume that they will move around together.
	linkTarget := absCurrent

	parentDir := filepath.Dir(absNew)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil && os.IsExist(err) {
		return fmt.Errorf("failed to create parent directories leading to %s: %s", targetDir, err)
	}

	err = os.Symlink(linkTarget, absNew)
	if err == nil {
		// Success, then!
		return nil
	}

	// If we get down here then symlinking failed and we need a deep copy
	// instead.
	err = copydir.CopyDir(absNew, absCurrent)
	if err != nil {
		return fmt.Errorf("failed to either symlink or copy %s to %s: %s", absCurrent, absNew, err)
	}

	// If we got here then apparently our copy succeeded, so we're done.
	return nil
}
