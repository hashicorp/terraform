package discovery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	getter "github.com/hashicorp/go-getter"
	multierror "github.com/hashicorp/go-multierror"
)

// Releases are located by parsing the html listing from releases.hashicorp.com.
//
// The URL for releases follows the pattern:
//    https://releases.hashicorp.com/terraform-provider-name/<x.y.z>/terraform-provider-name_<x.y.z>_<os>_<arch>.<ext>
//
// The plugin protocol version will be saved with the release and returned in
// the header X-TERRAFORM_PROTOCOL_VERSION.

const protocolVersionHeader = "x-terraform-protocol-version"

var releaseHost = "https://releases.hashicorp.com"

var httpClient = cleanhttp.DefaultClient()

// Plugins are referred to by the short name, but all URLs and files will use
// the full name prefixed with terraform-<plugin_type>-
func providerName(name string) string {
	return "terraform-provider-" + name
}

// providerVersionsURL returns the path to the released versions directory for the provider:
// https://releases.hashicorp.com/terraform-provider-name/
func providerVersionsURL(name string) string {
	return releaseHost + "/" + providerName(name) + "/"
}

// providerURL returns the full path to the provider file, using the current OS
// and ARCH:
// .../terraform-provider-name_<x.y.z>/terraform-provider-name_<x.y.z>_<os>_<arch>.<ext>
func providerURL(name, version string) string {
	fileName := fmt.Sprintf("%s_%s_%s_%s.zip", providerName(name), version, runtime.GOOS, runtime.GOARCH)
	u := fmt.Sprintf("%s%s/%s", providerVersionsURL(name), version, fileName)
	return u
}

// An Installer maintains a local cache of plugins by downloading plugins
// from an online repository.
type Installer interface {
	Get(name string, req Constraints) (PluginMeta, error)
	PurgeUnused(used map[string]PluginMeta) (removed PluginMetaSet, err error)
}

// ProviderInstaller is an Installer implementation that knows how to
// download Terraform providers from the official HashiCorp releases service
// into a local directory. The files downloaded are compliant with the
// naming scheme expected by FindPlugins, so the target directory of a
// provider installer can be used as one of several plugin discovery sources.
type ProviderInstaller struct {
	Dir string

	PluginProtocolVersion uint
}

func (i *ProviderInstaller) Get(provider string, req Constraints) (PluginMeta, error) {
	versions, err := listProviderVersions(provider)
	// TODO: return multiple errors
	if err != nil {
		return PluginMeta{}, err
	}

	if len(versions) == 0 {
		return PluginMeta{}, fmt.Errorf("no plugins found for provider %q", provider)
	}

	versions = allowedVersions(versions, req)
	if len(versions) == 0 {
		return PluginMeta{}, fmt.Errorf("no version of %q available that fulfills constraints %s", provider, req)
	}

	// sort them newest to oldest
	Versions(versions).Sort()

	// take the first matching plugin we find
	for _, v := range versions {
		url := providerURL(provider, v.String())
		log.Printf("[DEBUG] fetching provider info for %s version %s", provider, v)
		if checkPlugin(url, i.PluginProtocolVersion) {
			log.Printf("[DEBUG] getting provider %q version %q at %s", provider, v, url)
			err := getter.Get(i.Dir, url)
			if err != nil {
				return PluginMeta{}, err
			}

			// Find what we just installed
			// (This is weird, because go-getter doesn't directly return
			//  information about what was extracted, and we just extracted
			//  the archive directly into a shared dir here.)
			log.Printf("[DEBUG] looking for the %s %s plugin we just installed", provider, v)
			metas := FindPlugins("provider", []string{i.Dir})
			log.Printf("all plugins found %#v", metas)
			metas, _ = metas.ValidateVersions()
			metas = metas.WithName(provider).WithVersion(v)
			log.Printf("filtered plugins %#v", metas)
			if metas.Count() == 0 {
				// This should never happen. Suggests that the release archive
				// contains an executable file whose name doesn't match the
				// expected convention.
				return PluginMeta{}, fmt.Errorf(
					"failed to find installed provider %s %s; this is a bug in Terraform and should be reported",
					provider, v,
				)
			}

			if metas.Count() > 1 {
				// This should also never happen, and suggests that a
				// particular version was re-released with a different
				// executable filename. We consider releases as immutable, so
				// this is an error.
				return PluginMeta{}, fmt.Errorf(
					"multiple plugins installed for %s %s; this is a bug in Terraform and should be reported",
					provider, v,
				)
			}

			// By now we know we have exactly one meta, and so "Newest" will
			// return that one.
			return metas.Newest(), nil
		}

		log.Printf("[INFO] incompatible ProtocolVersion for %s version %s", provider, v)
	}

	return PluginMeta{}, fmt.Errorf("no versions of %q compatible with the plugin ProtocolVersion", provider)
}

func (i *ProviderInstaller) PurgeUnused(used map[string]PluginMeta) (PluginMetaSet, error) {
	purge := make(PluginMetaSet)

	present := FindPlugins("provider", []string{i.Dir})
	for meta := range present {
		chosen, ok := used[meta.Name]
		if !ok {
			purge.Add(meta)
		}
		if chosen.Path != meta.Path {
			purge.Add(meta)
		}
	}

	removed := make(PluginMetaSet)
	var errs error
	for meta := range purge {
		path := meta.Path
		err := os.Remove(path)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf(
				"failed to remove unused provider plugin %s: %s",
				path, err,
			))
		} else {
			removed.Add(meta)
		}
	}

	return removed, errs
}

// Return the plugin version by making a HEAD request to the provided url
func checkPlugin(url string, pluginProtocolVersion uint) bool {
	resp, err := httpClient.Head(url)
	if err != nil {
		log.Printf("[ERROR] error fetching plugin headers: %s", err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		log.Println("[ERROR] non-200 status fetching plugin headers:", resp.Status)
		return false
	}

	proto := resp.Header.Get(protocolVersionHeader)
	if proto == "" {
		log.Printf("[WARNING] missing %s from: %s", protocolVersionHeader, url)
		return false
	}

	protoVersion, err := strconv.Atoi(proto)
	if err != nil {
		log.Printf("[ERROR] invalid ProtocolVersion: %s", proto)
		return false
	}

	return protoVersion == int(pluginProtocolVersion)
}

var errVersionNotFound = errors.New("version not found")

// take the list of available versions for a plugin, and filter out those that
// don't fit the constraints.
func allowedVersions(available []Version, required Constraints) []Version {
	var allowed []Version

	for _, v := range available {
		if required.Allows(v) {
			allowed = append(allowed, v)
		}
	}

	return allowed
}

// list the version available for the named plugin
func listProviderVersions(name string) ([]Version, error) {
	versions, err := listPluginVersions(providerVersionsURL(name))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions for provider %q: %s", name, err)
	}
	return versions, nil
}

// return a list of the plugin versions at the given URL
func listPluginVersions(url string) ([]Version, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("[ERROR] failed to fetch plugin versions from %s\n%s\n%s", url, resp.Status, body)
		return nil, errors.New(resp.Status)
	}

	body, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	names := []string{}

	// all we need to do is list links on the directory listing page that look like plugins
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			c := n.FirstChild
			if c != nil && c.Type == html.TextNode && strings.HasPrefix(c.Data, "terraform-") {
				names = append(names, c.Data)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(body)

	return versionsFromNames(names), nil
}

// parse the list of directory names into a sorted list of available versions
func versionsFromNames(names []string) []Version {
	var versions []Version
	for _, name := range names {
		parts := strings.SplitN(name, "_", 2)
		if len(parts) == 2 && parts[1] != "" {
			v, err := VersionStr(parts[1]).Parse()
			if err != nil {
				// filter invalid versions scraped from the page
				log.Printf("[WARN] invalid version found for %q: %s", name, err)
				continue
			}

			versions = append(versions, v)
		}
	}

	return versions
}
