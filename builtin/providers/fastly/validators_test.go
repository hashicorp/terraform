package fastly

import "testing"

func TestValidateS3FormatVersion(t *testing.T) {
	validVersions := []uint{
		1,
		2,
	}
	for _, v := range validVersions {
		_, errors := validateS3FormatVersion(v, "format_version")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid format version: %q", v, errors)
		}
	}

	invalidVersions := []uint{
		0,
		3,
		4,
		5,
	}
	for _, v := range invalidVersions {
		_, errors := validateS3FormatVersion(v, "format_version")
		if len(errors) != 1 {
			t.Fatalf("%q should not be a valid format version", v)
		}
	}
}
