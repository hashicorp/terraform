package artifactory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Artifact defines the structure of an artifact
type Artifact struct {
	Info   FileInfo
	Client *Client
}

// ArtifactProperties represents a set of properties for an Artifact
type ArtifactProperties map[string][]string

// AQLProperties represents the json used to compose an AQL query
type AQLProperties struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// FileInfo represents the json returned by the artifactory API for a file
type FileInfo struct {
	URI          string             `json:"uri"`
	DownloadURI  string             `json:"downloadUri"`
	Repo         string             `json:"repo"`
	Path         string             `json:"path"`
	RemoteURL    string             `json:"remoteUrl,omitempty"`
	Created      string             `json:"created"`
	CreatedBy    string             `json:"createdBy"`
	LastModified string             `json:"lastModified"`
	ModifiedBy   string             `json:"modifiedBy"`
	LastUpdated  string             `json:"lastUpdated"`
	Size         string             `json:"size"`
	MimeType     string             `json:"mimeType"`
	Properties   ArtifactProperties `json:"properties"`
	Checksums    struct {
		MD5    string `json:"md5"`
		SHA1   string `json:"sha1"`
		SHA256 string `json:"sha256"`
	} `json:"checksums"`
	OriginalChecksums struct {
		MD5    string `json:"md5"`
		SHA1   string `json:"sha1"`
		SHA256 string `json:"sha256"`
	} `json:"originalChecksums,omitempty"`
}

// AQLResults represents the json returned from artifactory for an AQL query
type AQLResults struct {
	Results []AQLFileInfo `json:"results"`
	Range   struct {
		StartPos int `json:"start_pos"`
		EndPos   int `json:"end_pos"`
		Total    int `json:"total"`
		Limit    int `json:"limit"`
	} `json:"range"`
}

// AQLFileInfo represents the json returned from artifactory for an individual file result in an AQL query
type AQLFileInfo struct {
	Repo         string          `json:"repo,omitempty"`
	Path         string          `json:"path,omitempty"`
	Name         string          `json:"name,omitempty"`
	Type         string          `json:"type,omitempty"`
	Created      string          `json:"created,omitempty"`
	CreatedBy    string          `json:"created_by,omitempty"`
	Modified     string          `json:"modified,omitempty"`
	ModifiedBy   string          `json:"modified_by,omitempty"`
	Depth        int             `json:"depth,omitempty"`
	Size         int64           `json:"size,omitempty"`
	Properties   []AQLProperties `json:"properties,omitempty"`
	ActualMD5    string          `json:"actual_md5,omitempty"`
	ActualSHA1   string          `json:"actual_sha1,omitempty"`
	OriginalSHA1 string          `json:"original_sha1,omitempty"`
}

// Download downloads an artifact
func (c *Artifact) Download() ([]byte, error) {
	return c.Client.RetrieveArtifact(c.Info.Repo, c.Info.Path)
}

// Delete deletes an artifact
func (c *Artifact) Delete() error {
	_, err := c.Client.DeleteArtifact(c.Info.Repo, c.Info.Path)
	return err
}

// GetFileInfo returns the details about an artifact
func (c *Client) GetFileInfo(path string) (a Artifact, err error) {
	a.Client = c
	var res FileInfo
	d, err := c.HTTPRequest(Request{
		Verb: "GET",
		Path: "/api/storage/" + path,
	})
	if err != nil {
		return a, err
	}
	e := json.Unmarshal(d, &res)
	if e != nil {
		return a, e
	}
	a.Info = res
	return a, nil
}

// DeleteArtifact deletes the named artifact from the provided repo
func (c *Client) DeleteArtifact(repo, path string) ([]byte, error) {
	return c.HTTPRequest(Request{
		Verb: "DELETE",
		Path: "/" + repo + "/" + path,
	})

}

// RetrieveArtifact downloads the named artifact from the provided repo
func (c *Client) RetrieveArtifact(repo string, path string) ([]byte, error) {
	return c.HTTPRequest(Request{
		Verb: "GET",
		Path: "/" + repo + "/" + path,
	})
}

// DeployArtifact deploys the named artifact to the provided repo
func (c *Client) DeployArtifact(repoKey string, filename string, path string, properties map[string]string) (CreatedStorageItem, error) {
	var res CreatedStorageItem
	var fileProps []string
	finalURL := "/" + repoKey + "/"
	if &path != nil {
		finalURL = finalURL + path
	}
	baseFile := filepath.Base(filename)
	finalURL = finalURL + "/" + baseFile
	if len(properties) > 0 {
		finalURL = finalURL + ";"
		for k, v := range properties {
			fileProps = append(fileProps, k+"="+v)
		}
		finalURL = finalURL + strings.Join(fileProps, ";")
	}
	data, err := os.Open(filename)
	if err != nil {
		return res, err
	}
	defer func() { _ = data.Close() }()
	d, err := c.HTTPRequest(Request{
		Verb: "PUT",
		Path: finalURL,
		Body: data,
	})
	if err != nil {
		return res, err
	}
	e := json.Unmarshal(d, &res)
	return res, e
}
