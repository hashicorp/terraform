// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package acctest

import (
	"regexp"
	"testing"
)

func TestRandIpAddress(t *testing.T) {
	testCases := []struct {
		s           string
		expected    *regexp.Regexp
		expectedErr string
	}{
		{
			s:        "1.1.1.1/32",
			expected: regexp.MustCompile(`^1\.1\.1\.1$`),
		},
		{
			s:        "10.0.0.0/8",
			expected: regexp.MustCompile(`^10\.\d{1,3}\.\d{1,3}\.\d{1,3}$`),
		},
		{
			s:           "0.0.0.0/0",
			expectedErr: "CIDR range is too large: 32",
		},
		{
			s:        "449d:e5f1:14b1:ddf3:8525:7e9e:4a0d:4a82/128",
			expected: regexp.MustCompile(`^449d:e5f1:14b1:ddf3:8525:7e9e:4a0d:4a82$`),
		},
		{
			s:        "2001:db8::/112",
			expected: regexp.MustCompile(`^2001:db8::[[:xdigit:]]{1,4}$`),
		},
		{
			s:           "2001:db8::/64",
			expectedErr: "CIDR range is too large: 64",
		},
		{
			s:           "abcdefg",
			expectedErr: "invalid CIDR address: abcdefg",
		},
	}

	for i, tc := range testCases {
		v, err := RandIpAddress(tc.s)
		if err != nil {
			msg := err.Error()
			if tc.expectedErr == "" {
				t.Fatalf("expected test case %d to succeed but got error %q, ", i, msg)
			}
			if msg != tc.expectedErr {
				t.Fatalf("expected test case %d to fail with %q but got %q", i, tc.expectedErr, msg)
			}
		} else if !tc.expected.MatchString(v) {
			t.Fatalf("expected test case %d to return %q but got %q", i, tc.expected, v)
		}
	}
}
