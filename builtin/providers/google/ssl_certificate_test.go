package google

import (
	"testing"
)

func TestCanonicalizeCertUrl(t *testing.T) {
	cases := map[string]struct {
		Input    string
		Expected string
		Error    bool
	}{
		"Full URL": {
			Input:    "https://www.googleapis.com/compute/v1/projects/rls-ngi-dev/global/sslCertificates/my-cert",
			Expected: "https://www.googleapis.com/compute/v1/projects/rls-ngi-dev/global/sslCertificates/my-cert",
		},
		"Partial URL": {
			Input:    "projects/rls-ngi-dev/global/sslCertificates/my-cert",
			Expected: "https://www.googleapis.com/compute/v1/projects/rls-ngi-dev/global/sslCertificates/my-cert",
		},
		"Invalid URL": {
			Input: "my-cert",
			Error: true,
		},
	}

	for key, tc := range cases {
		got, err := canonicalizeCertUrl(tc.Input)
		if !tc.Error {
			if err != nil {
				t.Fatalf("[%s] Unexpected error happend: %s", key, err)
			}
			if got != tc.Expected {
				t.Fatalf("[%s] Expected '%#v', but got '%#v'", key, tc.Expected, got)
			}
		} else {
			if err == nil {
				t.Fatalf("[%s] Expected error to happen, but got '%#v' without any errors", key, got)
			}
		}
	}
}
