package remote

import (
	"os"
	"testing"
)

func TestArtifactoryClient_impl(t *testing.T) {
	var _ Client = new(ArtifactoryClient)
}

func TestArtifactoryFactory(t *testing.T) {
	for _, envVar := range []string{
		"ARTIFACTORY_TOKEN",
		"ARTIFACTORY_URL",
		"ARTIFACTORY_USERNAME",
		"ARTIFACTORY_PASSWORD",
	} {
		_ = os.Unsetenv(envVar)
	}
	testCases := map[string]struct {
		config     map[string]string
		shouldfail bool
		authmethod string
	}{
		"valid with basic": {
			shouldfail: false,
			config: map[string]string{
				"url":      "http://artifactory.local",
				"repo":     "terraform-repo",
				"subpath":  "myproject",
				"username": "test",
				"password": "testpass",
			},
			authmethod: "basic",
		},
		"valid with token": {
			shouldfail: false,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"repo":    "terraform-repo",
				"subpath": "myproject",
				"token":   "abcdefg",
			},
			authmethod: "token",
		},
		"invalid with no creds": {
			shouldfail: true,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"repo":    "terraform-repo",
				"subpath": "myproject",
			},
		},
		"invalid with no url": {
			shouldfail: true,
			config: map[string]string{
				"repo":    "terraform-repo",
				"subpath": "myproject",
				"token":   "abcdefg",
			},
		},
		"invalid with no repo": {
			shouldfail: true,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"subpath": "myproject",
				"token":   "abcdefg",
			},
		},
		"invalid with no subpath": {
			shouldfail: true,
			config: map[string]string{
				"url":   "http://artifactory.local",
				"repo":  "terraform-repo",
				"token": "abcdefg",
			},
		},
	}
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			client, err := artifactoryFactory(v.config)
			if v.shouldfail {
				if err == nil {
					t.Fatalf("test should throw an error")
				}
				if client != nil {
					t.Fatalf("client should be nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("error should be nil")
			}
			ac := client.(*ArtifactoryClient)
			if ac.nativeClient == nil {
				t.Fatalf("native client should not be nil")
			}
			if ac.nativeClient.Config.AuthMethod != v.authmethod {
				t.Fatalf("authmethod should match")
			}
		})
	}
}

func TestArtifactoryFactoryFromEnv(t *testing.T) {
	testCases := map[string]struct {
		config     map[string]string
		envVars    map[string]string
		shouldfail bool
		authmethod string
	}{
		"valid with basic from env": {
			shouldfail: false,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"repo":    "terraform-repo",
				"subpath": "myproject",
			},
			envVars: map[string]string{
				"ARTIFACTORY_USERNAME": "username",
				"ARTIFACTORY_PASSWORD": "password",
			},
			authmethod: "basic",
		},
		"valid with token from env": {
			shouldfail: false,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"repo":    "terraform-repo",
				"subpath": "myproject",
			},
			envVars: map[string]string{
				"ARTIFACTORY_TOKEN": "abcdef",
			},
			authmethod: "token",
		},
		"valid with url from env": {
			shouldfail: false,
			config: map[string]string{
				"repo":     "terraform-repo",
				"subpath":  "myproject",
				"username": "test",
				"password": "testpassword",
			},
			envVars: map[string]string{
				"ARTIFACTORY_URL": "http://artifactory.local",
			},
			authmethod: "basic",
		},
		"invalid with no creds in env": {
			shouldfail: true,
			config: map[string]string{
				"url":     "http://artifactory.local",
				"repo":    "terraform-repo",
				"subpath": "myproject",
			},
		},
	}
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			for _, envVar := range []string{
				"ARTIFACTORY_TOKEN",
				"ARTIFACTORY_URL",
				"ARTIFACTORY_USERNAME",
				"ARTIFACTORY_PASSWORD",
			} {
				_ = os.Unsetenv(envVar)
			}
			for k, v := range v.envVars {
				err := os.Setenv(k, v)
				if err != nil {
					t.Fatalf("should not throw an error setting env: %s", err.Error())
				}
			}
			client, err := artifactoryFactory(v.config)
			if v.shouldfail {
				if err == nil {
					t.Fatalf("test should throw an error")
				}
				if client != nil {
					t.Fatalf("client should be nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("error should be nil but got %s", err.Error())
			}
			ac := client.(*ArtifactoryClient)
			if ac.nativeClient == nil {
				t.Fatalf("native client should not be nil")
			}
			if ac.nativeClient.Config.AuthMethod != v.authmethod {
				t.Fatalf("authmethod should match")
			}
		})
	}
}
