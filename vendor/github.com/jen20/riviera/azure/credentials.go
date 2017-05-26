package azure

const (
	defaultResourceManagerEndpoint = "https://management.azure.com"
	defaultActiveDirectoryEndpoint = "https://login.microsoftonline.com"
)

type AzureResourceManagerCredentials struct {
	ClientID       string
	ClientSecret   string
	TenantID       string
	SubscriptionID string

	// can be overridden for non public clouds
	ResourceManagerEndpoint string
	ActiveDirectoryEndpoint string
}
