// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcs

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/zclconf/go-cty/cty"
)

func TestBackendConfig_encryptionKey(t *testing.T) {
	preCheckEnvironmentVariables(t)
	// Cannot use t.Parallel as t.SetEnv used

	// This function is required because the key input is changed internally in the backend's code.
	// This function is a quick way to help us get an expected value for tests, but ideally in future
	// the test and the code under test will use a reusable function to avoid logic duplication.
	expectedValue := func(key string) []byte {
		if key == "" {
			return nil
		}

		v, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			t.Fatalf("error in test setup: %s", err.Error())
		}
		return v
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
			want: expectedValue(""),
		},

		"set in config only": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": encryptionKey,
			},
			want: expectedValue(encryptionKey),
		},

		"set in config and GOOGLE_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": encryptionKey,
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2, // Different
			},
			want: expectedValue(encryptionKey),
		},

		"set in GOOGLE_ENCRYPTION_KEY only": {
			config: map[string]interface{}{
				"bucket": "foobar",
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2,
			},
			want: expectedValue(encryptionKey2),
		},

		"set in config as empty string and in GOOGLE_ENCRYPTION_KEY": {
			config: map[string]interface{}{
				"bucket":         "foobar",
				"encryption_key": "",
			},
			envs: map[string]string{
				"GOOGLE_ENCRYPTION_KEY": encryptionKey2,
			},
			want: expectedValue(encryptionKey2),
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
	preCheckEnvironmentVariables(t)
	// Cannot use t.Parallel() due to t.Setenv

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

func TestBackendEncryptionKeyEmptyConflict(t *testing.T) {
	// This test is for the edge case where encryption_key and
	// kms_encryption_key are both set in the configuration but set to empty
	// strings. The "SDK-like" helpers treat unset as empty string, so
	// we need an extra rule to catch them both being set to empty string
	// directly inside the configuration, and this test covers that
	// special case.
	//
	// The following assumes that the validation check we're testing will, if
	// failing, always block attempts to reach any real GCP services, and so
	// this test should be fine to run without an acceptance testing opt-in.

	// This test is for situations where these environment variables are not set.
	t.Setenv("GOOGLE_ENCRYPTION_KEY", "")
	t.Setenv("GOOGLE_KMS_ENCRYPTION_KEY", "")

	backend := New()
	schema := backend.ConfigSchema()
	rawVal := cty.ObjectVal(map[string]cty.Value{
		"bucket": cty.StringVal("fake-placeholder"),

		// These are both empty strings but should still be considered as
		// set when we enforce teh rule that they can't both be set at once.
		"encryption_key":     cty.StringVal(""),
		"kms_encryption_key": cty.StringVal(""),
	})
	// The following mimicks how the terraform_remote_state data source
	// treats its "config" argument, which is a realistic situation where
	// we take an arbitrary object and try to force it to conform to the
	// backend's schema.
	configVal, err := schema.CoerceValue(rawVal)
	if err != nil {
		t.Fatalf("unexpected coersion error: %s", err)
	}
	configVal, diags := backend.PrepareConfig(configVal)
	if diags.HasErrors() {
		t.Fatalf("unexpected PrepareConfig error: %s", diags.Err().Error())
	}

	configDiags := backend.Configure(configVal)
	if !configDiags.HasErrors() {
		t.Fatalf("unexpected success; want error")
	}
	gotErr := configDiags.Err().Error()
	wantErr := `can't set both encryption_key and kms_encryption_key`
	if !strings.Contains(gotErr, wantErr) {
		t.Errorf("wrong error\ngot: %s\nwant substring: %s", gotErr, wantErr)
	}
}
