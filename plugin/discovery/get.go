package discovery

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	getter "github.com/hashicorp/go-getter"
)

const releasesURL = "https://releases.hashicorp.com/"

var httpClient = cleanhttp.DefaultClient()

// pluginURL generates URLs to lookup the versions of a plugin, or the file path.
//
// The URL for releases follows the pattern:
//    https://releases.hashicorp.com/terraform-providers/terraform-provider-name/ +
//        terraform-provider-name_<x.y.z>/terraform-provider-name_<x.y.z>_<os>_<arch>.<ext>
type pluginURL struct {
	// the base url to search for releases
	releases string
	// The name prefix common to all plugins of this type.
	// This is either `terraform-provider` or `terraform-provisioner`.
	baseName string
}

// releasesURL returns the top level directory for all plugins of this type
func (p pluginURL) releasesURL() string {
	// the top level directory is the plural form of the plugin type
	return p.releases + p.baseName + "s"
}

// versionsURL returns the url to the directory to list available versionsURL for this plugin
func (p pluginURL) versionsURL(name string) string {
	return fmt.Sprintf("%s/%s-%s", p.releasesURL(), p.baseName, name)
}

// fileURL returns the full path to a plugin based on the plugin name,
// version, GOOS and GOARCH.
func (p pluginURL) fileURL(name, version string) string {
	releasesDir := fmt.Sprintf("%s-%s_%s/", p.baseName, name, version)
	fileName := fmt.Sprintf("%s-%s_%s_%s_%s.zip", p.baseName, name, version, runtime.GOOS, runtime.GOARCH)
	return fmt.Sprintf("%s/%s/%s", p.versionsURL(name), releasesDir, fileName)
}

var providersURL = pluginURL{
	releases: releasesURL,
	baseName: "terraform-provider",
}

var provisionersURL = pluginURL{
	releases: releasesURL,
	baseName: "terraform-provisioners",
}

// GetProvider fetches a provider plugin based on the version constraints, and
// copies it to the dst directory.
//
// TODO: verify checksum and signature
func GetProvider(dst, provider string, req Constraints) error {
	versions, err := listProviderVersions(provider)
	// TODO: return multiple errors
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return fmt.Errorf("no plugins found for provider %q", provider)
	}

	versions = filterProtocolVersions(provider, versions)

	if len(versions) == 0 {
		return fmt.Errorf("no versions of %q compatible with the plugin ProtocolVersion", provider)
	}

	version, err := newestVersion(versions, req)
	if err != nil {
		return fmt.Errorf("no version of %q available that fulfills constraints %s", provider, req)
	}

	url := providersURL.fileURL(provider, version.String())

	log.Printf("[DEBUG] getting provider %q version %q at %s", provider, version, url)
	return getter.Get(dst, url)
}

// Remove available versions that don't have the correct plugin protocol version.
// TODO: stop checking older versions if the protocol version is too low
func filterProtocolVersions(provider string, versions []Version) []Version {
	var compatible []Version
	for _, v := range versions {
		log.Printf("[DEBUG] fetching provider info for %s version %s", provider, v)
		url := providersURL.fileURL(provider, v.String())
		resp, err := httpClient.Head(url)
		if err != nil {
			log.Printf("[ERROR] error fetching plugin headers: %s", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Println("[ERROR] non-200 status fetching plugin headers:", resp.Status)
			continue
		}

		proto := resp.Header.Get("X-TERRAFORM_PROTOCOL_VERSION")
		if proto == "" {
			log.Println("[WARNING] missing X-TERRAFORM_PROTOCOL_VERSION from:", url)
			continue
		}

		protoVersion, err := strconv.Atoi(proto)
		if err != nil {
			log.Println("[ERROR] invalid ProtocolVersion: %s", proto)
			continue
		}

		// FIXME: this shouldn't be hardcoded
		if protoVersion != 4 {
			log.Printf("[INFO] incompatible ProtocolVersion %d from %s version %s", protoVersion, provider, v)
			continue
		}

		compatible = append(compatible, v)
	}

	return compatible
}

var errVersionNotFound = errors.New("version not found")

// take the list of available versions for a plugin, and the required
// Constraints, and return the latest available version that satisfies the
// constraints.
func newestVersion(available []Version, required Constraints) (Version, error) {
	var latest Version
	found := false

	for _, v := range available {
		if required.Allows(v) {
			if !found {
				latest = v
				found = true
				continue
			}

			if v.NewerThan(latest) {
				latest = v
			}
		}
	}

	if !found {
		return latest, errVersionNotFound
	}
	return latest, nil
}

// list the version available for the named plugin
func listProviderVersions(name string) ([]Version, error) {
	versions, err := listPluginVersions(providersURL.versionsURL(name))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions for provider %q: %s", name, err)
	}
	return versions, nil
}

func listProvisionerVersions(name string) ([]Version, error) {
	versions, err := listPluginVersions(provisionersURL.versionsURL(name))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions for provisioner %q: %s", name, err)
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

	Versions(versions).Sort()
	return versions
}
