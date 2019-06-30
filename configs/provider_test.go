package configs

import (
	"io/ioutil"
	"testing"
)

func TestProviderReservedNames(t *testing.T) {
	src, err := ioutil.ReadFile("testdata/invalid-files/provider-reserved.tf")
	if err != nil {
		t.Fatal(err)
	}
	parser := testParser(map[string]string{
		"config.tf": string(src),
	})
	_, diags := parser.LoadConfigFile("config.tf")

	assertExactDiagnostics(t, diags, []string{
		`config.tf:10,3-8: Reserved argument name in provider block; The provider argument name "count" is reserved for use by Terraform in a future version.`,
		`config.tf:11,3-13: Reserved argument name in provider block; The provider argument name "depends_on" is reserved for use by Terraform in a future version.`,
		`config.tf:12,3-11: Reserved argument name in provider block; The provider argument name "for_each" is reserved for use by Terraform in a future version.`,
		`config.tf:14,3-12: Reserved block type name in provider block; The block type name "lifecycle" is reserved for use by Terraform in a future version.`,
		`config.tf:15,3-9: Reserved block type name in provider block; The block type name "locals" is reserved for use by Terraform in a future version.`,
		`config.tf:13,3-9: Reserved argument name in provider block; The provider argument name "source" is reserved for use by Terraform in a future version.`,
	})
}
