package chef

import "fmt"

// CookbookService  is the service for interacting with chef server cookbooks endpoint
type CookbookService struct {
	client *Client
}

// CookbookItem represents a object of cookbook file data
type CookbookItem struct {
	Url         string `json:"url,omitempty"`
	Path        string `json:"path,omitempty"`
	Name        string `json:"name,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	Specificity string `json:"specificity,omitempty"`
}

// CookbookListResult is the summary info returned by chef-api when listing
// http://docs.opscode.com/api_chef_server.html#cookbooks
type CookbookListResult map[string]CookbookVersions

// CookbookRecipesResult is the summary info returned by chef-api when listing
// http://docs.opscode.com/api_chef_server.html#cookbooks-recipes
type CookbookRecipesResult []string

// CookbookVersions is the data container returned from the chef server when listing all cookbooks
type CookbookVersions struct {
	Url      string            `json:"url,omitempty"`
	Versions []CookbookVersion `json:"versions,omitempty"`
}

// CookbookVersion is the data for a specific cookbook version
type CookbookVersion struct {
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

// CookbookMeta represents a Golang version of cookbook metadata
type CookbookMeta struct {
	Name            string                 `json:"cookbook_name,omitempty"`
	Version         string                 `json:"version,omitempty"`
	Description     string                 `json:"description,omitempty"`
	LongDescription string                 `json:"long_description,omitempty"`
	Maintainer      string                 `json:"maintainer,omitempty"`
	MaintainerEmail string                 `json:"maintainer_email,omitempty"`
	License         string                 `json:"license,omitempty"`
	Platforms       map[string]string      `json:"platforms,omitempty"`
	Depends         map[string]string      `json:"dependencies,omitempty"`
	Reccomends      map[string]string      `json:"recommendations,omitempty"`
	Suggests        map[string]string      `json:"suggestions,omitempty"`
	Conflicts       map[string]string      `json:"conflicting,omitempty"`
	Provides        map[string]string      `json:"providing,omitempty"`
	Replaces        map[string]string      `json:"replacing,omitempty"`
	Attributes      map[string]interface{} `json:"attributes,omitempty"` // this has a format as well that could be typed, but blargh https://github.com/lob/chef/blob/master/cookbooks/apache2/metadata.json
	Groupings       map[string]interface{} `json:"groupings,omitempty"`  // never actually seen this used.. looks like it should be map[string]map[string]string, but not sure http://docs.opscode.com/essentials_cookbook_metadata.html
	Recipes         map[string]string      `json:"recipes,omitempty"`
}

// Cookbook represents the native Go version of the deserialized api cookbook
type Cookbook struct {
	CookbookName string         `json:"cookbook_name"`
	Name         string         `json:"name"`
	Version      string         `json:"version,omitempty"`
	ChefType     string         `json:"chef_type,omitempty"`
	Frozen       bool           `json:"frozen?,omitempty"`
	JsonClass    string         `json:"json_class,omitempty"`
	Files        []CookbookItem `json:"files,omitempty"`
	Templates    []CookbookItem `json:"templates,omitempty"`
	Attributes   []CookbookItem `json:"attributes,omitempty"`
	Recipes      []CookbookItem `json:"recipes,omitempty"`
	Definitions  []CookbookItem `json:"definitions,omitempty"`
	Libraries    []CookbookItem `json:"libraries,omitempty"`
	Providers    []CookbookItem `json:"providers,omitempty"`
	Resources    []CookbookItem `json:"resources,omitempty"`
	RootFiles    []CookbookItem `json:"templates,omitempty"`
	Metadata     CookbookMeta   `json:"metadata,omitempty"`
}

// String makes CookbookListResult implement the string result
func (c CookbookListResult) String() (out string) {
	for k, v := range c {
		out += fmt.Sprintf("%s => %s\n", k, v.Url)
		for _, i := range v.Versions {
			out += fmt.Sprintf(" * %s\n", i.Version)
		}
	}
	return out
}

// versionParams assembles a querystring for the chef api's  num_versions
// This is used to restrict the number of versions returned in the reponse
func versionParams(path, numVersions string) string {
	if numVersions == "0" {
		numVersions = "all"
	}

	// need to optionally add numVersion args to the request
	if len(numVersions) > 0 {
		path = fmt.Sprintf("%s?num_versions=%s", path, numVersions)
	}
	return path
}

// Get retruns a CookbookVersion for a specific cookbook
//  GET /cookbooks/name
func (c *CookbookService) Get(name string) (data CookbookVersion, err error) {
	path := fmt.Sprintf("cookbooks/%s", name)
	err = c.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// GetAvailable returns the versions of a coookbook available on a server
func (c *CookbookService) GetAvailableVersions(name, numVersions string) (data CookbookListResult, err error) {
	path := versionParams(fmt.Sprintf("cookbooks/%s", name), numVersions)
	err = c.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// GetVersion fetches a specific version of a cookbooks data from the server api
//   GET /cookbook/foo/1.2.3
//   GET /cookbook/foo/_latest
//   Chef API docs: http://docs.opscode.com/api_chef_server.html#id5
func (c *CookbookService) GetVersion(name, version string) (data Cookbook, err error) {
	url := fmt.Sprintf("cookbooks/%s/%s", name, version)
	c.client.magicRequestDecoder("GET", url, nil, &data)
	return
}

// ListVersions lists the cookbooks available on the server limited to numVersions
//   Chef API docs: http://docs.opscode.com/api_chef_server.html#id2
func (c *CookbookService) ListAvailableVersions(numVersions string) (data CookbookListResult, err error) {
	path := versionParams("cookbooks", numVersions)
	err = c.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// ListAllRecipes lists the names of all recipes in the most recent cookbook versions
//   Chef API docs: https://docs.chef.io/api_chef_server.html#id31
func (c *CookbookService) ListAllRecipes() (data CookbookRecipesResult, err error) {
	path := "cookbooks/_recipes"
	err = c.client.magicRequestDecoder("GET", path, nil, &data)
	return
}

// List returns a CookbookListResult with the latest versions of cookbooks available on the server
func (c *CookbookService) List() (CookbookListResult, error) {
	return c.ListAvailableVersions("")
}

// DeleteVersion removes a version of a cook from a server
func (c *CookbookService) Delete(name, version string) (err error) {
	path := fmt.Sprintf("cookbooks/%s/%s", name, version)
	err = c.client.magicRequestDecoder("DELETE", path, nil, nil)
	return
}
