package artifactory

import (
	"encoding/json"
	"fmt"
)

// Repo represents the json response from Artifactory describing a repository
type Repo struct {
	Key         string `json:"key"`
	Rtype       string `json:"type"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// RepoConfig represents a repo config
type RepoConfig interface {
	MimeType() string
}

// GenericRepoConfig represents the common json of a repo response from artifactory
type GenericRepoConfig struct {
	Key                          string   `json:"key,omitempty"`
	RClass                       string   `json:"rclass"`
	PackageType                  string   `json:"packageType,omitempty"`
	Description                  string   `json:"description,omitempty"`
	Notes                        string   `json:"notes,omitempty"`
	IncludesPattern              string   `json:"includesPattern,omitempty"`
	ExcludesPattern              string   `json:"excludesPattern,omitempty"`
	LayoutRef                    string   `json:"repoLayoutRef,omitempty"`
	HandleReleases               bool     `json:"handleReleases,omitempty"`
	HandleSnapshots              bool     `json:"handleSnapshots,omitempty"`
	MaxUniqueSnapshots           int      `json:"maxUniqueSnapshots,omitempty"`
	SuppressPomConsistencyChecks bool     `json:"suppressPomConsistencyChecks,omitempty"`
	BlackedOut                   bool     `json:"blackedOut,omitempty"`
	PropertySets                 []string `json:"propertySets,omitempty"`
}

// MimeType returns the MimeType of a GenericRepoConfig
func (r GenericRepoConfig) MimeType() string {
	return ""
}

// LocalRepoConfig represents a local repo type in artifactory
type LocalRepoConfig struct {
	GenericRepoConfig

	DebianTrivialLayout     bool   `json:"debianTrivialLayout,omitempty"`
	ChecksumPolicyType      string `json:"checksumPolicyType,omitempty"`
	MaxUniqueTags           int    `json:"maxUniqueTags,omitempty"`
	SnapshotVersionBehavior string `json:"snapshotVersionBehavior,omitempty"`
	ArchiveBrowsingEnabled  bool   `json:"archiveBrowsingEnabled,omitempty"`
	CalculateYumMetadata    bool   `json:"calculateYumMetadata,omitempty"`
	YumRootDepth            int    `json:"yumRootDepth,omitempty"`
	DockerAPIVersion        string `json:"dockerApiVersion,omitempty"`
	EnableFileListsIndexing bool   `json:"enableFileListsIndexing,omitempty"`
}

// MimeType returns the MimeType for a local repo in artifactory
func (r LocalRepoConfig) MimeType() string {
	return LocalRepoMimeType
}

// RemoteRepoConfig represents a remote repo in artifactory
type RemoteRepoConfig struct {
	GenericRepoConfig

	URL                               string `json:"url"`
	Username                          string `json:"username,omitempty"`
	Password                          string `json:"password,omitempty"`
	Proxy                             string `json:"proxy,omitempty"`
	RemoteRepoChecksumPolicyType      string `json:"remoteRepoChecksumPolicyType,omitempty"`
	HardFail                          bool   `json:"hardFail,omitempty"`
	Offline                           bool   `json:"offline,omitempty"`
	StoreArtifactsLocally             bool   `json:"storeArtifactsLocally,omitempty"`
	SocketTimeoutMillis               int    `json:"socketTimeoutMillis,omitempty"`
	LocalAddress                      string `json:"localAddress,omitempty"`
	RetrivialCachePeriodSecs          int    `json:"retrievalCachePeriodSecs,omitempty"`
	FailedRetrievalCachePeriodSecs    int    `json:"failedRetrievalCachePeriodSecs,omitempty"`
	MissedRetrievalCachePeriodSecs    int    `json:"missedRetrievalCachePeriodSecs,omitempty"`
	UnusedArtifactsCleanupEnabled     bool   `json:"unusedArtifactCleanupEnabled,omitempty"`
	UnusedArtifactsCleanupPeriodHours int    `json:"unusedArtifactCleanupPeriodHours,omitempty"`
	FetchJarsEagerly                  bool   `json:"fetchJarsEagerly,omitempty"`
	FetchSourcesEagerly               bool   `json:"fetchSourcesEagerly,omitempty"`
	ShareConfiguration                bool   `json:"shareConfiguration,omitempty"`
	SynchronizeProperties             bool   `json:"synchronizeProperties,omitempty"`
	BlockMismatchingMimeTypes         bool   `json:"blockMismatchingMimeTypes,omitempty"`
	AllowAnyHostAuth                  bool   `json:"allowAnyHostAuth,omitempty"`
	EnableCookieManagement            bool   `json:"enableCookieManagement,omitempty"`
	BowerRegistryURL                  string `json:"bowerRegistryUrl,omitempty"`
	VcsType                           string `json:"vcsType,omitempty"`
	VcsGitProvider                    string `json:"vcsGitProvider,omitempty"`
	VcsGitDownloader                  string `json:"vcsGitDownloader,omitempty"`
	ClientTLSCertificate              string `json:"clientTlsCertificate,omitempty"`
}

// MimeType returns the mimetype of a remote repo
func (r RemoteRepoConfig) MimeType() string {
	return RemoteRepoMimeType
}

// VirtualRepoConfig represents a virtual repo in artifactory
type VirtualRepoConfig struct {
	GenericRepoConfig

	Repositories                                  []string `json:"repositories"`
	DebianTrivialLayout                           bool     `json:"debianTrivialLayout,omitempty"`
	ArtifactoryRequestsCanRetrieveRemoteArtifacts bool     `json:"artifactoryRequestsCanRetrieveRemoteArtifacts,omitempty"`
	KeyPair                                       string   `json:"keyPair,omitempty"`
	PomRepositoryReferencesCleanupPolicy          string   `json:"pomRepositoryReferencesCleanupPolicy,omitempty"`
	DefaultDeploymentRepo                         string   `json:"defaultDeploymentRepo,omitempty"`
}

// MimeType returns the mimetype for a virtual repo in artifactory
func (r VirtualRepoConfig) MimeType() string {
	return VirtualRepoMimeType
}

// GetRepos returns all repos of the provided type
func (c *Client) GetRepos(rtype string) ([]Repo, error) {
	o := make(map[string]string)
	if rtype != "all" {
		o["type"] = rtype
	}
	var dat []Repo
	d, e := c.Get("/api/repositories", o)
	if e != nil {
		return dat, e
	}
	err := json.Unmarshal(d, &dat)
	if err != nil {
		return dat, err
	}
	return dat, e
}

// GetRepo returns the named repo
func (c *Client) GetRepo(key string, q map[string]string) (RepoConfig, error) {
	var dat GenericRepoConfig
	d, err := c.Get("/api/repositories/"+key, q)
	if err != nil {
		return dat, err
	}
	err = json.Unmarshal(d, &dat)
	if err != nil {
		return dat, err
	}
	switch dat.RClass {
	case "local":
		var cdat LocalRepoConfig
		_ = json.Unmarshal(d, &cdat)
		return cdat, nil
	case "remote":
		var cdat RemoteRepoConfig
		_ = json.Unmarshal(d, &cdat)
		return cdat, nil
	case "virtual":
		var cdat VirtualRepoConfig
		_ = json.Unmarshal(d, &cdat)
		return cdat, nil
	default:
		fmt.Printf("fallthrough to default\n")
		return dat, nil
	}
}

// CreateRepo creates the named repo
func (c *Client) CreateRepo(key string, r RepoConfig, q map[string]string) error {
	j, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = c.Put("/api/repositories/"+key, j, q)
	return err
}

// UpdateRepo updates the named repo
func (c *Client) UpdateRepo(key string, r RepoConfig, q map[string]string) error {
	j, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = c.Post("/api/repositories/"+key, j, q)
	return err
}
