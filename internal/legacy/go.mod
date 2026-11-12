module github.com/hashicorp/terraform/internal/legacy

replace github.com/hashicorp/terraform => ../..

go 1.26.11

require (
	github.com/hashicorp/terraform v0.0.0-00010101000000-000000000000
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mitchellh/reflectwalk v1.0.2
	github.com/zclconf/go-cty v1.18.1
)

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.5.0 // indirect
	github.com/hashicorp/terraform-svchost v0.2.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
)
