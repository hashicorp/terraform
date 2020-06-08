package awsbase

type Config struct {
	AccessKey                   string
	AssumeRoleARN               string
	AssumeRoleDurationSeconds   int
	AssumeRoleExternalID        string
	AssumeRolePolicy            string
	AssumeRolePolicyARNs        []string
	AssumeRoleSessionName       string
	AssumeRoleTags              map[string]string
	AssumeRoleTransitiveTagKeys []string
	CallerDocumentationURL      string
	CallerName                  string
	CredsFilename               string
	DebugLogging                bool
	IamEndpoint                 string
	Insecure                    bool
	MaxRetries                  int
	Profile                     string
	Region                      string
	SecretKey                   string
	SkipCredsValidation         bool
	SkipMetadataApiCheck        bool
	SkipRequestingAccountId     bool
	StsEndpoint                 string
	Token                       string
	UserAgentProducts           []*UserAgentProduct
}

type UserAgentProduct struct {
	Extra   []string
	Name    string
	Version string
}
