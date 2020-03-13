package providercache

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	getter "github.com/hashicorp/go-getter"

	"github.com/hashicorp/terraform/httpclient"
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

func installFromLocalDir(ctx context.Context, sourceDir string, targetDir string) error {
	return fmt.Errorf("installFromLocalDir not yet implemented")
}
