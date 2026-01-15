module github.com/hashicorp/terraform/internal/backend/remote-state/s3

go 1.25

require (
	github.com/aws/aws-sdk-go-v2 v1.39.2
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.9
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.22
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.50.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.88.3
	github.com/aws/smithy-go v1.23.0
	github.com/google/go-cmp v0.7.0
	github.com/hashicorp/aws-sdk-go-base/v2 v2.0.0-beta.67
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-uuid v1.0.3
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/hashicorp/terraform v0.0.0-00010101000000-000000000000
	github.com/zclconf/go-cty v1.16.4
)

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-versions v1.0.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.12 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.47.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.8.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.38.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.6 // indirect
	github.com/bmatcuk/doublestar v1.1.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/go-slug v0.18.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/terraform-plugin-log v0.9.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.4.0 // indirect
	github.com/hashicorp/terraform-svchost v0.2.0 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/zclconf/go-cty-yaml v1.1.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/tools v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
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
