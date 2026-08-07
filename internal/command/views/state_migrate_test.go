// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestNewStateMigrate_LogProviderVersionSuccess(t *testing.T) {
	const verifiedChecksum = 0
	const officialProvider = 1
	const noKey = ""

	t.Run("no auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		var authResult *getproviders.PackageAuthenticationResult = nil

		smView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (unauthenticated)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("verified checksum auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		authResult := getproviders.NewPackageAuthenticationResult(verifiedChecksum, noKey)

		smView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (verified checksum)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
	t.Run("official provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(officialProvider, key)

		smView.LogProviderVersionSuccess(p, ver, authResult)

		// Assert output
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by HashiCorp)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_LogProviderVersionSuccessWithKeyID(t *testing.T) {
	const partnerProvider = 2

	t.Run("partner provider auth result - human view", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		p := addrs.MustParseProviderSourceString("hashicorp/test")
		ver := getproviders.MustParseVersion("1.2.3")
		key := "key-id-123"
		authResult := getproviders.NewPackageAuthenticationResult(partnerProvider, key)

		smView.LogProviderVersionSuccessWithKeyID(p, ver, authResult, key)

		// Assert output - human
		output := done(t)
		expectedOutput := "- Installed hashicorp/test v1.2.3 (signed by a HashiCorp partner, key ID key-id-123)\n"
		if output.Stdout() != expectedOutput {
			t.Fatalf("expected %q, got %q", expectedOutput, output.Stdout())
		}
	})
}

func TestNewStateMigrate_Spacer_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.Spacer()

	// Assert output
	output := done(t)

	// We cannot simply assert no output as the JSON view logs the version message on initialization
	// Splitting on \n when there's only the version log will get an array of the log and an empty string.
	// If there are more logs there'll be >2 elements.
	if x := strings.Split(output.Stdout(), "\n"); len(x) != 2 {
		t.Fatalf("expected no additional output after version message, got: %s", output.Stdout())
	}
}

func TestNewStateMigrate_LogStateMigrationErrored_human(t *testing.T) {
	t.Run("migration error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringMigration, source, destination)

		// Assert output
		output := done(t)
		expectedOutputSnippets := []string{
			`Failed to migrate state from backend "local" to state store "test_store"`,
		}
		for _, snippet := range expectedOutputSnippets {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
	t.Run("provider lockfile error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringLockfile, source, destination)

		// Assert output
		output := done(t)
		expectedOutputSnippets := []string{
			`Finished migrating state from backend "local" to state store "test_store", but an error occurred before Terraform was finished.`,
		}
		for _, snippet := range expectedOutputSnippets {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
	t.Run("backend state file error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := NewStateMigrate(arguments.ViewHuman, view)

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringBackendStateFile, source, destination)

		// Assert output
		output := done(t)
		expectedOutputSnippets := []string{
			`Finished migrating state from backend "local" to state store "test_store", but an error occurred before Terraform was finished.`,
		}
		for _, snippet := range expectedOutputSnippets {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
}

func TestNewStateMigrate_LogInteractiveApproval_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogInteractiveApproval()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`approved by the user`, // @message
		`"@module":"terraform.ui"`,
		`"type":"state_store_provider_interactive_approval"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogInteractiveRejection_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogInteractiveRejection()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`rejected by the user`, // @message
		`"@module":"terraform.ui"`,
		`"type":"state_store_provider_interactive_rejection"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogAutomaticApproval_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogAutomaticApproval()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`approved automatically`, // @message
		`"@module":"terraform.ui"`,
		`"type":"state_store_provider_automatic_approval"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogStateMigrationStart_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	source := `backend "local"`
	destination := `state store "test_store"`
	smView.LogStateMigrationStart(source, destination)

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Migrating state from backend \"local\" to state store \"test_store\"..."`,
		`"@module":"terraform.ui"`,
		`"type":"migration_start"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogStateMigrationComplete_json(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	smView := StateMigrateJSON{view: NewJSONView(view)}

	smView.LogStateMigrationComplete()

	// Assert output
	output := done(t)
	expectedOutputFields := []string{
		`"@level":"info"`,
		`"@message":"Migration complete"`,
		`"@module":"terraform.ui"`,
		`"type":"migration_complete"`,
	}
	for _, snippet := range expectedOutputFields {
		if !strings.Contains(output.Stdout(), snippet) {
			t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
		}
	}
}

func TestNewStateMigrate_LogStateMigrationErrored_json(t *testing.T) {
	t.Run("migration error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := StateMigrateJSON{view: NewJSONView(view)}

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringMigration, source, destination)

		// Assert output
		output := done(t)
		expectedOutputFields := []string{
			`"@level":"info"`,
			`Failed to migrate state from`, // @message
			`"@module":"terraform.ui"`,
			`"failure_mode":"error_during_migration"`, // custom field
			`"type":"migration_errored"`,
		}
		for _, snippet := range expectedOutputFields {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
	t.Run("provider lockfile error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := StateMigrateJSON{view: NewJSONView(view)}

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringLockfile, source, destination)

		// Assert output
		output := done(t)
		expectedOutputFields := []string{
			`"@level":"info"`,
			`Finished migrating state`,                        // @message
			`an error occurred before Terraform was finished`, // @message
			`"@module":"terraform.ui"`,
			`"failure_mode":"error_updating_provider_lockfile"`, // custom field
			`"type":"migration_errored"`,
		}
		for _, snippet := range expectedOutputFields {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
	t.Run("backend state file error", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		smView := StateMigrateJSON{view: NewJSONView(view)}

		source := `backend "local"`
		destination := `state store "test_store"`

		smView.LogStateMigrationErrored(DuringBackendStateFile, source, destination)

		// Assert output
		output := done(t)
		expectedOutputFields := []string{
			`"@level":"info"`,
			`Finished migrating state`,                        // @message
			`an error occurred before Terraform was finished`, // @message
			`"@module":"terraform.ui"`,
			`"failure_mode":"error_updating_workdir_state"`, // custom field
			`"type":"migration_errored"`,
		}
		for _, snippet := range expectedOutputFields {
			if !strings.Contains(output.Stdout(), snippet) {
				t.Fatalf("output didn't include expected snippet:\n expected: %s\n got:\n %s", snippet, output.Stdout())
			}
		}
	})
}
