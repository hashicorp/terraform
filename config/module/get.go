package module

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/hashicorp/go-getter"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

// GetMode is an enum that describes how modules are loaded.
//
// GetModeLoad says that modules will not be downloaded or updated, they will
// only be loaded from the storage.
//
// GetModeGet says that modules can be initially downloaded if they don't
// exist, but otherwise to just load from the current version in storage.
//
// GetModeUpdate says that modules should be checked for updates and
// downloaded prior to loading. If there are no updates, we load the version
// from disk, otherwise we download first and then load.
type GetMode byte

const (
	GetModeNone GetMode = iota
	GetModeGet
	GetModeUpdate
)

// GetCopy is the same as Get except that it downloads a copy of the
// module represented by source.
//
// This copy will omit and dot-prefixed files (such as .git/, .hg/) and
// can't be updated on its own.
func GetCopy(dst, src string) error {
	// Create the temporary directory to do the real Get to
	tmpDir, err := ioutil.TempDir("", "tf")
	if err != nil {
		return err
	}
	// FIXME: This isn't completely safe. Creating and removing our temp path
	//        exposes where to race to inject files.
	if err := os.RemoveAll(tmpDir); err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Get to that temporary dir
	if err := getter.Get(tmpDir, src); err != nil {
		return err
	}

	// Make sure the destination exists
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Copy to the final location
	return copyDir(dst, tmpDir)
}

const (
	registryAPI   = "https://registry.terraform.io/v1/modules"
	xTerraformGet = "X-Terraform-Get"
)

var detectors = []getter.Detector{
	new(getter.GitHubDetector),
	new(getter.BitBucketDetector),
	new(getter.S3Detector),
	new(registryDetector),
	new(getter.FileDetector),
}

// these prefixes can't be registry IDs
// "http", "../", "./", "/", "getter::", etc
var skipRegistry = regexp.MustCompile(`^(http|[.]{1,2}/|/|[A-Za-z0-9]+::)`).MatchString

// registryDetector implements getter.Detector to detect Terraform Registry modules.
// If a path looks like a registry module identifier, attempt to locate it in
// the registry. If it's not found, pass it on in case it can be found by
// other means.
type registryDetector struct {
	// override the default registry URL
	api string

	client *http.Client
}

func (d registryDetector) Detect(src, _ string) (string, bool, error) {
	// the namespace can't start with "http", a relative or absolute path, or
	// contain a go-getter "forced getter"
	if skipRegistry(src) {
		return "", false, nil
	}

	// there are 3 parts to a registry ID
	if len(strings.Split(src, "/")) != 3 {
		return "", false, nil
	}

	return d.lookupModule(src)
}

// Lookup the module in the registry.
func (d registryDetector) lookupModule(src string) (string, bool, error) {
	if d.api == "" {
		d.api = registryAPI
	}

	if d.client == nil {
		d.client = cleanhttp.DefaultClient()
	}

	// src is already partially validated in Detect. We know it's a path, and
	// if it can be parsed as a URL we will hand it off to the registry to
	// determine if it's truly valid.
	resp, err := d.client.Get(fmt.Sprintf("%s/%s/download", d.api, src))
	if err != nil {
		return "", false, fmt.Errorf("error looking up module %q: %s", src, err)
	}
	defer resp.Body.Close()

	// there should be no body, but save it for logging
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", false, fmt.Errorf("error reading response body from registry: %s", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		// OK
	case http.StatusNotFound:
		return "", false, fmt.Errorf("module %q not found in registry", src)
	default:
		// anything else is an error:
		return "", false, fmt.Errorf("error getting download location for %q: %s resp:%s", src, resp.Status, body)
	}

	// the download location is in the X-Terraform-Get header
	location := resp.Header.Get(xTerraformGet)
	if location == "" {
		return "", false, fmt.Errorf("failed to get download URL for %q: %s resp:%s", src, resp.Status, body)
	}

	return location, true, nil
}
