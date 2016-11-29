package docker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDockerRegistryImage() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDockerRegistryImageRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"sha256_digest": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDockerRegistryImageRead(d *schema.ResourceData, meta interface{}) error {
	pullOpts := parseImageOptions(d.Get("name").(string))

	// Use the official Docker Hub if a registry isn't specified
	if pullOpts.Registry == "" {
		pullOpts.Registry = "registry.hub.docker.com"
	} else {
		// Otherwise, filter the registry name out of the repo name
		pullOpts.Repository = strings.Replace(pullOpts.Repository, pullOpts.Registry+"/", "", 1)
	}

	// Docker prefixes 'library' to official images in the path; 'consul' becomes 'library/consul'
	if !strings.Contains(pullOpts.Repository, "/") {
		pullOpts.Repository = "library/" + pullOpts.Repository
	}

	if pullOpts.Tag == "" {
		pullOpts.Tag = "latest"
	}

	digest, err := getImageDigest(pullOpts.Registry, pullOpts.Repository, pullOpts.Tag, "", "")

	if err != nil {
		return fmt.Errorf("Got error when attempting to fetch image version from registry: %s", err)
	}

	d.SetId(digest)
	d.Set("sha256_digest", digest)

	return nil
}

func getImageDigest(registry, image, tag, username, password string) (string, error) {
	client := http.DefaultClient

	req, err := http.NewRequest("GET", "https://"+registry+"/v2/"+image+"/manifests/"+tag, nil)

	if err != nil {
		return "", fmt.Errorf("Error creating registry request: %s", err)
	}

	if username != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)

	if err != nil {
		return "", fmt.Errorf("Error during registry request: %s", err)
	}

	switch resp.StatusCode {
	// Basic auth was valid or not needed
	case http.StatusOK:
		return resp.Header.Get("Docker-Content-Digest"), nil

	// Either OAuth is required or the basic auth creds were invalid
	case http.StatusUnauthorized:
		if strings.HasPrefix(resp.Header.Get("www-authenticate"), "Bearer") {
			auth := parseAuthHeader(resp.Header.Get("www-authenticate"))
			params := url.Values{}
			params.Set("service", auth["service"])
			params.Set("scope", auth["scope"])
			tokenRequest, err := http.NewRequest("GET", auth["realm"]+"?"+params.Encode(), nil)

			if err != nil {
				return "", fmt.Errorf("Error creating registry request: %s", err)
			}

			if username != "" {
				tokenRequest.SetBasicAuth(username, password)
			}

			tokenResponse, err := client.Do(tokenRequest)

			if err != nil {
				return "", fmt.Errorf("Error during registry request: %s", err)
			}

			if tokenResponse.StatusCode != http.StatusOK {
				return "", fmt.Errorf("Got bad response from registry: " + tokenResponse.Status)
			}

			body, err := ioutil.ReadAll(tokenResponse.Body)
			if err != nil {
				return "", fmt.Errorf("Error reading response body: %s", err)
			}

			token := &TokenResponse{}
			err = json.Unmarshal(body, token)
			if err != nil {
				return "", fmt.Errorf("Error parsing OAuth token response: %s", err)
			}

			req.Header.Set("Authorization", "Bearer "+token.Token)
			digestResponse, err := client.Do(req)

			if err != nil {
				return "", fmt.Errorf("Error during registry request: %s", err)
			}

			if digestResponse.StatusCode != http.StatusOK {
				return "", fmt.Errorf("Got bad response from registry: " + digestResponse.Status)
			}

			return digestResponse.Header.Get("Docker-Content-Digest"), nil
		} else {
			return "", fmt.Errorf("Bad credentials: " + resp.Status)
		}

	// Some unexpected status was given, return an error
	default:
		return "", fmt.Errorf("Got bad response from registry: " + resp.Status)
	}
}

type TokenResponse struct {
	Token string
}

// Parses key/value pairs from a WWW-Authenticate header
func parseAuthHeader(header string) map[string]string {
	parts := strings.SplitN(header, " ", 2)
	parts = strings.Split(parts[1], ",")
	opts := make(map[string]string)

	for _, part := range parts {
		vals := strings.SplitN(part, "=", 2)
		key := vals[0]
		val := strings.Trim(vals[1], "\", ")
		opts[key] = val
	}

	return opts
}
