module github.com/hashicorp/terraform/internal/backend/remote-state/s3

go 1.25

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.22
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.43.2
	github.com/aws/aws-sdk-go-v2/service/s3 v1.80.1
	github.com/aws/smithy-go v1.22.3
	github.com/google/go-cmp v0.7.0
	github.com/hashicorp/aws-sdk-go-base/v2 v2.0.0-beta.65
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/hashicorp/terraform v0.0.0-00010101000000-000000000000
	github.com/zclconf/go-cty v1.16.3
)

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-versions v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.15 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.68 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.42.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.10.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.34.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.38.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.20 // indirect
	github.com/bmatcuk/doublestar v1.1.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-slug v0.16.7 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/terraform-plugin-log v0.9.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.3.0 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/zclconf/go-cty-yaml v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.61.0 // indirect
	go.opentelemetry.io/otel v1.36.0 // indirect
	go.opentelemetry.io/otel/metric v1.36.0 // indirect
	go.opentelemetry.io/otel/trace v1.36.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
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

replace github.com/hashicorp/terraform/internal => ../../..

replace github.com/hashicorp/terraform => ../../../..
