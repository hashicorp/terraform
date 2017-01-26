package cloudflare

import "testing"

func TestValidateRecordType(t *testing.T) {
	validTypes := []string{
		"A",
		"AAAA",
		"CNAME",
		"TXT",
		"SRV",
		"LOC",
		"MX",
		"NS",
		"SPF",
	}
	for _, v := range validTypes {
		_, errors := validateRecordType(v, "type")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid record type: %q", v, errors)
		}
	}

	invalidTypes := []string{
		"a",
		"cName",
		"txt",
		"SRv",
		"foo",
		"bar",
	}
	for _, v := range invalidTypes {
		_, errors := validateRecordType(v, "type")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid record type", v)
		}
	}
}

func TestValidateRecordName(t *testing.T) {
	validNames := map[string]string{
		"A":    "192.168.0.1",
		"AAAA": "2001:0db8:0000:0042:0000:8a2e:0370:7334",
	}

	for k, v := range validNames {
		if err := validateRecordName(k, v); err != nil {
			t.Fatalf("%q should be a valid name for type %q: %v", v, k, err)
		}
	}

	invalidNames := map[string]string{
		"A":    "terraform.io",
		"AAAA": "192.168.0.1",
	}
	for k, v := range invalidNames {
		if err := validateRecordName(k, v); err == nil {
			t.Fatalf("%q should be an invalid name for type %q", v, k)
		}
	}
}
