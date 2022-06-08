package cloud

import (
	"testing"

	"github.com/hashicorp/terraform/internal/configs"
	legacy "github.com/hashicorp/terraform/internal/legacy/terraform"
)

func TestDetectConfigChangeType(t *testing.T) {
	tests := map[string]struct {
		stateType            string
		configType           string
		localStates          bool
		want                 ConfigChangeMode
		wantInvolvesCloud    bool
		wantIsCloudMigration bool
	}{
		"init cloud": {
			``, `cloud`, false,
			ConfigChangeInPlace,
			true, false,
		},
		"reinit cloud": {
			`cloud`, `cloud`, false,
			ConfigChangeInPlace,
			true, false,
		},
		"migrate default local to cloud with existing local state": {
			``, `cloud`, true,
			ConfigMigrationIn,
			true, true,
		},
		"migrate local to cloud": {
			`local`, `cloud`, false,
			ConfigMigrationIn,
			true, true,
		},
		"migrate remote to cloud": {
			`local`, `cloud`, false,
			ConfigMigrationIn,
			true, true,
		},
		"migrate cloud to local": {
			`cloud`, `local`, false,
			ConfigMigrationOut,
			true, true,
		},
		"migrate cloud to remote": {
			`cloud`, `remote`, false,
			ConfigMigrationOut,
			true, true,
		},
		"migrate cloud to default local": {
			`cloud`, ``, false,
			ConfigMigrationOut,
			true, true,
		},

		// Various other cases can potentially be valid (decided by the
		// Terraform CLI layer) but are irrelevant for Cloud mode purposes.
		"init default local": {
			``, ``, false,
			ConfigChangeIrrelevant,
			false, false,
		},
		"init default local with existing local state": {
			``, ``, true,
			ConfigChangeIrrelevant,
			false, false,
		},
		"init remote backend": {
			``, `remote`, false,
			ConfigChangeIrrelevant,
			false, false,
		},
		"init remote backend with existing local state": {
			``, `remote`, true,
			ConfigChangeIrrelevant,
			false, false,
		},
		"reinit remote backend": {
			`remote`, `remote`, false,
			ConfigChangeIrrelevant,
			false, false,
		},
		"migrate local to remote backend": {
			`local`, `remote`, false,
			ConfigChangeIrrelevant,
			false, false,
		},
		"migrate remote to default local": {
			`remote`, ``, false,
			ConfigChangeIrrelevant,
			false, false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var state *legacy.BackendState
			var config *configs.Backend
			if test.stateType != "" {
				state = &legacy.BackendState{
					Type: test.stateType,
					// everything else is irrelevant for our purposes here
				}
			}
			if test.configType != "" {
				config = &configs.Backend{
					Type: test.configType,
					// everything else is irrelevant for our purposes here
				}
			}
			got := DetectConfigChangeType(state, config, test.localStates)

			if got != test.want {
				t.Errorf(
					"wrong result\nstate type:   %s\nconfig type:  %s\nlocal states: %t\n\ngot:  %s\nwant: %s",
					test.stateType, test.configType, test.localStates,
					got, test.want,
				)
			}
			if got, want := got.InvolvesCloud(), test.wantInvolvesCloud; got != want {
				t.Errorf(
					"wrong InvolvesCloud result\ngot:  %t\nwant: %t",
					got, want,
				)
			}
			if got, want := got.IsCloudMigration(), test.wantIsCloudMigration; got != want {
				t.Errorf(
					"wrong IsCloudMigration result\ngot:  %t\nwant: %t",
					got, want,
				)
			}
		})
	}
}
