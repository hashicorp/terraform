package artifactory

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

type ArtifactChecksums struct {
	MD5  string `json:"md5"`
	SHA1 string `json:"sha1"`
}
