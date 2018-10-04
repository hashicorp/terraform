package artifactory

import (
	"encoding/json"
	"fmt"
	"strings"
)

// CreatedStorageItem represents a created storage item in artifactory
type CreatedStorageItem struct {
	URI               string            `json:"uri"`
	DownloadURI       string            `json:"downloadUri"`
	Repo              string            `json:"repo"`
	Created           string            `json:"created"`
	CreatedBy         string            `json:"createdBy"`
	Size              string            `json:"size"`
	MimeType          string            `json:"mimeType"`
	Checksums         ArtifactChecksums `json:"checksums"`
	OriginalChecksums ArtifactChecksums `json:"originalChecksums"`
}

// ArtifactChecksums represents the checksums for an artifact
type ArtifactChecksums struct {
	MD5  string `json:"md5"`
	SHA1 string `json:"sha1"`
}

// FileList represents a list of files
type FileList struct {
	URI     string         `json:"uri"`
	Created string         `json:"created"`
	Files   []FileListItem `json:"files"`
}

// FileListItem represents an item in a list of files
type FileListItem struct {
	URI          string `json:"uri"`
	Size         int    `json:"size"`
	LastModified string `json:"lastModified"`
	Folder       bool   `json:"folder"`
	SHA1         string `json:"sha1"`
}

// ItemProperties represents a set of properties for an item in artifactory
type ItemProperties struct {
	URI        string              `json:"uri"`
	Properties map[string][]string `json:"properties"`
}

// GetFileList lists all files in the specified repo
func (c *Client) GetFileList(repo string, path string) (FileList, error) {
	var fileList FileList

	d, err := c.Get(fmt.Sprintf("/api/storage/%s/%s?list&deep=1", repo, path), make(map[string]string))
	if err != nil {
		return fileList, err
	}

	err = json.Unmarshal(d, &fileList)
	return fileList, err
}

// GetItemProperties returns the properties for a specific item
func (c *Client) GetItemProperties(repo string, path string) (ItemProperties, error) {
	var itemProps ItemProperties

	d, err := c.Get(fmt.Sprintf("/api/storage/%s/%s?properties", repo, path), make(map[string]string))
	if err != nil {
		return itemProps, err
	}

	err = json.Unmarshal(d, &itemProps)
	return itemProps, err
}

// SetItemProperties attaches properties to an item (file or folder)
func (c *Client) SetItemProperties(repo string, path string, properties map[string][]string) error {
	var propertyString string
	var index int
	for k, v := range properties {
		index++
		if len(v) == 1 {
			propertyString = propertyString + fmt.Sprintf("%s=%s", k, v[0])
		} else {
			propertyString = propertyString + fmt.Sprintf("%s=[%s]", k, strings.Join(v, ","))
		}

		if index != len(properties) {
			propertyString = propertyString + ";"
		}
	}

	_, err := c.Put(fmt.Sprintf("/api/storage/%s/%s?properties=%s&recursive=1", repo, path, propertyString), nil, make(map[string]string))
	return err
}

// DeleteItemProperties deletes the specified properties from an item (file or folder)
func (c *Client) DeleteItemProperties(repo string, path string, properties []string) error {
	var propertyString string
	for _, v := range properties {
		propertyString = propertyString + fmt.Sprintf("%s,", v)
	}

	err := c.Delete(fmt.Sprintf("/api/storage/%s/%s?properties=%s&recursive=1", repo, path, propertyString))
	return err
}
