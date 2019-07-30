package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/svchost"
)

func TestHelperProgramCredentialsSource(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	program := filepath.Join(wd, "testdata/test-helper")
	t.Logf("testing with helper at %s", program)

	src := HelperProgramCredentialsSource(program)

	t.Run("happy path", func(t *testing.T) {
		creds, err := src.ForHost(svchost.Hostname("example.com"))
		if err != nil {
			t.Fatal(err)
		}
		if tokCreds, isTok := creds.(HostCredentialsToken); isTok {
			if got, want := string(tokCreds), "example-token"; got != want {
				t.Errorf("wrong token %q; want %q", got, want)
			}
		} else {
			t.Errorf("wrong type of credentials %T", creds)
		}
	})
	t.Run("no credentials", func(t *testing.T) {
		creds, err := src.ForHost(svchost.Hostname("nothing.example.com"))
		if err != nil {
			t.Fatal(err)
		}
		if creds != nil {
			t.Errorf("got credentials; want nil")
		}
	})
	t.Run("unsupported credentials type", func(t *testing.T) {
		creds, err := src.ForHost(svchost.Hostname("other-cred-type.example.com"))
		if err != nil {
			t.Fatal(err)
		}
		if creds != nil {
			t.Errorf("got credentials; want nil")
		}
	})
	t.Run("lookup error", func(t *testing.T) {
		_, err := src.ForHost(svchost.Hostname("fail.example.com"))
		if err == nil {
			t.Error("completed successfully; want error")
		}
	})
	t.Run("store happy path", func(t *testing.T) {
		err := src.StoreForHost(svchost.Hostname("example.com"), HostCredentialsToken("example-token"))
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("store error", func(t *testing.T) {
		err := src.StoreForHost(svchost.Hostname("fail.example.com"), HostCredentialsToken("example-token"))
		if err == nil {
			t.Error("completed successfully; want error")
		}
	})
	t.Run("forget happy path", func(t *testing.T) {
		err := src.ForgetForHost(svchost.Hostname("example.com"))
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("forget error", func(t *testing.T) {
		err := src.ForgetForHost(svchost.Hostname("fail.example.com"))
		if err == nil {
			t.Error("completed successfully; want error")
		}
	})
}
