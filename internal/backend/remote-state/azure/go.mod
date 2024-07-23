module github.com/hashicorp/terraform/internal/backend/remote-state/azure

go 1.22.5

require (
	github.com/Azure/azure-sdk-for-go v59.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.27
	github.com/hashicorp/go-azure-helpers v0.43.0
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/terraform v0.0.0-00010101000000-000000000000
	github.com/hashicorp/terraform/internal/legacy v0.0.0-00010101000000-000000000000
	github.com/manicminer/hamilton v0.44.0
	github.com/tombuildsstuff/giovanni v0.15.1
)

require (
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.20 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.4 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-versions v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-slug v0.15.0 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/hcl/v2 v2.20.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.2.3 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/manicminer/hamilton-autorest v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/zclconf/go-cty v1.14.4 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/mod v0.16.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/oauth2 v0.18.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.19.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/hashicorp/terraform/internal/backend/remote-state/azure => ../azure

replace github.com/hashicorp/terraform/internal/backend/remote-state/consul => ../consul

replace github.com/hashicorp/terraform/internal/backend/remote-state/cos => ../cos

replace github.com/hashicorp/terraform/internal/backend/remote-state/gcs => ../gcs

replace github.com/hashicorp/terraform/internal/backend/remote-state/kubernetes => ../kubernetes

replace github.com/hashicorp/terraform/internal/backend/remote-state/oss => ../oss

replace github.com/hashicorp/terraform/internal/backend/remote-state/pg => ../pg

replace github.com/hashicorp/terraform/internal/backend/remote-state/s3 => ../s3

replace github.com/hashicorp/terraform/internal/legacy => ../../../legacy

replace github.com/hashicorp/terraform => ../../../..
