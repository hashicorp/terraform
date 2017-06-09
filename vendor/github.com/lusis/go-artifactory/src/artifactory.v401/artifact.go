package artifactory

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Uri          string `json:"uri"`
	DownloadUri  string `json:"downloadUri"`
	Repo         string `json:"repo"`
	Path         string `json:"path"`
	RemoteUrl    string `json:"remoteUrl,omitempty"`
	Created      string `json:"created"`
	CreatedBy    string `json:"createdBy"`
	LastModified string `json:"lastModified"`
	ModifiedBy   string `json:"modifiedBy"`
	MimeType     string `json:"mimeType"`
	Size         string `json:"size"`
	Checksums    struct {
		SHA1 string `json:"sha1"`
		MD5  string `json:"md5"`
	} `json:"checksums"`
	OriginalChecksums struct {
		SHA1 string `json:"sha1"`
		MD5  string `json:"md5"`
	} `json:"originalChecksums,omitempty"`
}

func (c *ArtifactoryClient) DeployArtifact(repoKey string, filename string, path string, properties map[string]string) (CreatedStorageItem, error) {
	var res CreatedStorageItem
	var fileProps []string
	var finalUrl string
	finalUrl = "/" + repoKey + "/"
	if &path != nil {
		finalUrl = finalUrl + path
	}
	baseFile := filepath.Base(filename)
	finalUrl = finalUrl + "/" + baseFile
	if len(properties) > 0 {
		finalUrl = finalUrl + ";"
		for k, v := range properties {
			fileProps = append(fileProps, k+"="+v)
		}
		finalUrl = finalUrl + strings.Join(fileProps, ";")
	}
	data, err := os.Open(filename)
	if err != nil {
		return res, err
	}
	defer data.Close()
	b, _ := ioutil.ReadAll(data)
	d, err := c.Put(finalUrl, string(b), make(map[string]string))
	if err != nil {
		return res, err
	} else {
		e := json.Unmarshal(d, &res)
		if e != nil {
			return res, e
		} else {
			return res, nil
		}
	}
}
