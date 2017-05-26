package cloudflare

import "testing"

func TestValidateRecordType(t *testing.T) {
	validTypes := map[string]bool{
		"A":     true,
		"AAAA":  true,
		"CNAME": true,
		"TXT":   false,
		"SRV":   false,
		"LOC":   false,
		"MX":    false,
		"NS":    false,
		"SPF":   false,
	}
	for k, v := range validTypes {
		err := validateRecordType(k, v)
		if err != nil {
			t.Fatalf("%s should be a valid record type: %s", k, err)
		}
	}

	invalidTypes := map[string]bool{
		"a":     false,
		"cName": false,
		"txt":   false,
		"SRv":   false,
		"foo":   false,
		"bar":   false,
		"TXT":   true,
		"SRV":   true,
		"SPF":   true,
	}
	for k, v := range invalidTypes {
		if err := validateRecordType(k, v); err == nil {
			t.Fatalf("%s should be an invalid record type", k)
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
