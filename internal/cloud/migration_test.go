// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"testing"

	"github.com/hashicorp/terraform/internal/command/workdir"
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
			stateType:            ``,
			configType:           `cloud`,
			localStates:          false,
			want:                 ConfigChangeInPlace,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: false,
		},
		"reinit cloud": {
			stateType:            `cloud`,
			configType:           `cloud`,
			localStates:          false,
			want:                 ConfigChangeInPlace,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: false,
		},
		"migrate default local to cloud with existing local state": {
			stateType:            ``,
			configType:           `cloud`,
			localStates:          true,
			want:                 ConfigMigrationIn,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},
		"migrate local to cloud": {
			stateType:            `local`,
			configType:           `cloud`,
			localStates:          false,
			want:                 ConfigMigrationIn,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},
		"migrate remote to cloud": {
			stateType:            `local`,
			configType:           `cloud`,
			localStates:          false,
			want:                 ConfigMigrationIn,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},
		"migrate cloud to local": {
			stateType:            `cloud`,
			configType:           `local`,
			localStates:          false,
			want:                 ConfigMigrationOut,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},
		"migrate cloud to remote": {
			stateType:            `cloud`,
			configType:           `remote`,
			localStates:          false,
			want:                 ConfigMigrationOut,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},
		"migrate cloud to default local": {
			stateType:            `cloud`,
			configType:           ``,
			localStates:          false,
			want:                 ConfigMigrationOut,
			wantInvolvesCloud:    true,
			wantIsCloudMigration: true,
		},

		// Various other cases can potentially be valid (decided by the
		// Terraform CLI layer) but are irrelevant for Cloud mode purposes.
		"init default local": {
			stateType:            ``,
			configType:           ``,
			localStates:          false,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"init default local with existing local state": {
			stateType:            ``,
			configType:           ``,
			localStates:          true,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"init remote backend": {
			stateType:            ``,
			configType:           `remote`,
			localStates:          false,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"init remote backend with existing local state": {
			stateType:            ``,
			configType:           `remote`,
			localStates:          true,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"reinit remote backend": {
			stateType:            `remote`,
			configType:           `remote`,
			localStates:          false,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"migrate local to remote backend": {
			stateType:            `local`,
			configType:           `remote`,
			localStates:          false,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
		"migrate remote to default local": {
			stateType:            `remote`,
			configType:           ``,
			localStates:          false,
			want:                 ConfigChangeIrrelevant,
			wantInvolvesCloud:    false,
			wantIsCloudMigration: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var state *workdir.BackendStateFile
			if test.stateType != "" {
				state = &workdir.BackendStateFile{
					Backend: &workdir.BackendConfigState{
						Type: test.stateType,
						// everything else is irrelevant for our purposes here
					},
				}
			}

			got := DetectConfigChangeType(state, test.configType, test.localStates)

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
