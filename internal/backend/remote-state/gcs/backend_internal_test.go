// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcs

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
)

func TestBackendConfig_encryptionKey(t *testing.T) {
	// Cannot use t.Parallel as t.SetEnv used

	// TODO - add pre check that asserts ENVs for credentials are set when the test runs

	// getWantValue is required because the key input is changed internally in the backend's code
	// This function is a quick way to help us get a want value, but ideally in future the test and
	// the code under test will use a reusable function to avoid logic duplication.
	getWantValue := func(key string) []byte {
		var want []byte
		if key == "" {
			want = nil
		}
		if key != "" {
			var err error
			want, err = base64.StdEncoding.DecodeString(key)
			if err != nil {
				t.Fatalf("error in test setup: %s", err.Error())
			}
		}
		return want
	}

	cases := map[string]struct {
		config map[string]interface{}
		envs   map[string]string
		want   []byte
	}{
		"unset in config and ENVs": {
			config: map[string]interface{}{
				"bucket": "foobar",
			},
			want: getWantValue(""),
		},

		"set in config only": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": encryptionKey,
			},
			want: getWantValue(encryptionKey),
		},

		"set in config and GOOGLE_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": encryptionKey,
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2, // Different
			},
			want: getWantValue(encryptionKey),
		},

		"set in GOOGLE_ENCRYPTION_KEY only": {
			config: map[string]interface{}{
				"bucket": "foobar",
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2,
			},
			want: getWantValue(encryptionKey2),
		},

		"set in config as empty string and in GOOGLE_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": "",
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2,
			},
			want: getWantValue(encryptionKey2),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(tc.config))
			be := b.(*Backend)

			if !reflect.DeepEqual(be.encryptionKey, tc.want) {
				t.Fatalf("unexpected encryption_key value: wanted %v, got %v", tc.want, be.encryptionKey)
			}
		})
	}
}

func TestBackendConfig_kmsKey(t *testing.T) {
	t.Parallel()
	// TODO - add pre check that asserts ENVs for credentials are set when the test runs

	cases := map[string]struct {
		config map[string]interface{}
		envs   map[string]string
		want   string
	}{
		"unset in config and ENVs": {
			config: map[string]interface{}{
				"bucket": "foobar",
			},
		},

		"set in config only": {
			config: map[string]interface{}{
				"bucket":             "foobar",
				"kms_encryption_key": "value from config",
			},
			want: "value from config",
		},

		"set in config and GOOGLE_KMS_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":             "foobar",
				"kms_encryption_key": "value from config",
			},
			envs: map[string]string{
				"GOOGLE_KMS_ENCRYPTION_KEY": "value from GOOGLE_KMS_ENCRYPTION_KEY",
			},
			want: "value from config",
		},

		"set in GOOGLE_KMS_ENCRYPTION_KEY only": {
			config: map[string]interface{}{
				"bucket": "foobar",
			},
			envs: map[string]string{
				"GOOGLE_KMS_ENCRYPTION_KEY": "value from GOOGLE_KMS_ENCRYPTION_KEY",
			},
			want: "value from GOOGLE_KMS_ENCRYPTION_KEY",
		},

		"set in config as empty string and in GOOGLE_KMS_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":             "foobar",
				"kms_encryption_key": "",
			},
			envs: map[string]string{
				"GOOGLE_KMS_ENCRYPTION_KEY": "value from GOOGLE_KMS_ENCRYPTION_KEY",
			},
			want: "value from GOOGLE_KMS_ENCRYPTION_KEY",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(tc.config))
			be := b.(*Backend)

			if be.kmsKeyName != tc.want {
				t.Fatalf("unexpected kms_encryption_key value: wanted %v, got %v", tc.want, be.kmsKeyName)
			}
		})
	}
}

func TestStateFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prefix        string
		name          string
		wantStateFile string
		wantLockFile  string
	}{
		{"state", "default", "state/default.tfstate", "state/default.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
	}
	for _, c := range cases {
		b := &Backend{
			prefix: c.prefix,
		}

		if got := b.stateFile(c.name); got != c.wantStateFile {
			t.Errorf("stateFile(%q) = %q, want %q", c.name, got, c.wantStateFile)
		}

		if got := b.lockFile(c.name); got != c.wantLockFile {
			t.Errorf("lockFile(%q) = %q, want %q", c.name, got, c.wantLockFile)
		}
	}
}
