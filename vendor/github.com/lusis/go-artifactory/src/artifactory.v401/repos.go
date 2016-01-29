package artifactory

import (
	"encoding/json"
	"fmt"
)

type Repo struct {
	Key         string `json:"key"`
	Rtype       string `json:"type"`
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
}

type RepoConfig interface {
	MimeType() string
}

type GenericRepoConfig struct {
	Key                          string   `json:"key,omitempty"`
	RClass                       string   `json:"rclass"`
	PackageType                  string   `json:"packageType,omitempty"`
	Description                  string   `json:"description,omitempty"`
	Notes                        string   `json:"notes,omitempty"`
	IncludesPattern              string   `json:"includesPattern,omitempty"`
	ExcludesPattern              string   `json:"excludesPattern,omitempty"`
	HandleReleases               bool     `json:"handleReleases,omitempty"`
	HandleSnapshots              bool     `json:"handleSnapshots,omitempty"`
	MaxUniqueSnapshots           int      `json:"maxUniqueSnapshots,omitempty"`
	SuppressPomConsistencyChecks bool     `json:"supressPomConsistencyChecks,omitempty"`
	BlackedOut                   bool     `json:"blackedOut,omitempty"`
	PropertySets                 []string `json:"propertySets,omitempty"`
}

func (r GenericRepoConfig) MimeType() string {
	return ""
}

type LocalRepoConfig struct {
	GenericRepoConfig

	LayoutRef               string `json:"repoLayoutRef,omitempty"`
	DebianTrivialLayout     bool   `json:"debianTrivialLayout,omitempty"`
	ChecksumPolicyType      string `json:"checksumPolicyType,omitempty"`
	SnapshotVersionBehavior string `json:"snapshotVersionBehavior,omitempty"`
	ArchiveBrowsingEnabled  bool   `json:"archiveBrowsingEnabled,omitempty"`
	CalculateYumMetadata    bool   `json:"calculateYumMetadata,omitempty"`
	YumRootDepth            int    `json:"yumRootDepth,omitempty"`
}

func (r LocalRepoConfig) MimeType() string {
	return LOCAL_REPO_MIMETYPE
}

type RemoteRepoConfig struct {
	GenericRepoConfig

	Url                               string `json:"url"`
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
	ShareConfiguration                bool   `json:"shareConfiguration,omitempty"`
	SynchronizeProperties             bool   `json:"synchronizeProperties,omitempty"`
	AllowAnyHostAuth                  bool   `json:"allowAnyHostAuth,omitempty"`
	EnableCookieManagement            bool   `json:"enableCookieManagement,omitempty"`
	BowerRegistryUrl                  string `json:"bowerRegistryUrl,omitempty"`
	VcsType                           string `json:"vcsType,omitempty"`
	VcsGitProvider                    string `json:"vcsGitProvider,omitempty"`
	VcsGitDownloader                  string `json:"vcsGitDownloader,omitempty"`
}

func (r RemoteRepoConfig) MimeType() string {
	return REMOTE_REPO_MIMETYPE
}

type VirtualRepoConfig struct {
	GenericRepoConfig

	Repositories                                  []string `json:"repositories"`
	DebianTrivialLayout                           bool     `json:"debianTrivialLayout,omitempty"`
	ArtifactoryRequestsCanRetrieveRemoteArtifacts bool     `json:artifactoryRequestsCanRetrieveRemoteArtifacts,omitempty"`
	KeyPair                                       string   `json:"keyPair,omitempty"`
	PomRepositoryReferenceCleanupPolicy           string   `json:"pomRepositoryReferenceCleanupPolicy,omitempty"`
}

func (r VirtualRepoConfig) MimeType() string {
	return VIRTUAL_REPO_MIMETYPE
}

func (client *ArtifactoryClient) GetRepos(rtype string) ([]Repo, error) {
	o := make(map[string]string, 0)
	if rtype != "all" {
		o["type"] = rtype
	}
	var dat []Repo
	d, e := client.Get("/api/repositories", o)
	if e != nil {
		return dat, e
	} else {
		err := json.Unmarshal(d, &dat)
		if err != nil {
			return dat, err
		} else {
			return dat, e
		}
	}
}

func (client *ArtifactoryClient) GetRepo(key string) (RepoConfig, error) {
	o := make(map[string]string, 0)
	dat := new(GenericRepoConfig)
	d, e := client.Get("/api/repositories/"+key, o)
	if e != nil {
		return *dat, e
	} else {
		err := json.Unmarshal(d, &dat)
		if err != nil {
			return *dat, err
		} else {
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
	}
}
