module github.com/hashicorp/terraform/internal/legacy

replace github.com/hashicorp/terraform => ../..

go 1.23.3

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/google/go-cmp v0.6.0
	github.com/hashicorp/terraform v0.0.0-00010101000000-000000000000
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/mitchellh/reflectwalk v1.0.2
	github.com/zclconf/go-cty v1.16.0
)

require (
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-versions v1.0.2 // indirect
	github.com/hashicorp/go-slug v0.16.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.2.3 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	golang.org/x/mod v0.21.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.25.0 // indirect
)
