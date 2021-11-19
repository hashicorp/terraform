package cloud

import (
	"github.com/hashicorp/terraform/internal/configs"
	legacy "github.com/hashicorp/terraform/internal/legacy/terraform"
)

// Most of the logic for migrating into and out of "cloud mode" actually lives
// in the "command" package as part of the general backend init mechanisms,
// but we have some cloud-specific helper functionality here.

// ConfigChangeMode is a rough way to think about different situations that
// our backend change and state migration codepaths need to distinguish in
// the context of Cloud integration mode.
type ConfigChangeMode rune

//go:generate go run golang.org/x/tools/cmd/stringer -type ConfigChangeMode

const (
	// ConfigMigrationIn represents when the configuration calls for using
	// Cloud mode but the working directory state disagrees.
	ConfigMigrationIn ConfigChangeMode = '↘'

	// ConfigMigrationOut represents when the working directory state calls
	// for using Cloud mode but the working directory state disagrees.
	ConfigMigrationOut ConfigChangeMode = '↖'

	// ConfigChangeInPlace represents when both the working directory state
	// and the config call for using Cloud mode, and so there might be
	// (but won't necessarily be) cloud settings changing, but we don't
	// need to do any actual migration.
	ConfigChangeInPlace ConfigChangeMode = '↻'

	// ConfigChangeIrrelevant represents when the config and working directory
	// state disagree but neither calls for using Cloud mode, and so the
	// Cloud integration is not involved in dealing with this.
	ConfigChangeIrrelevant ConfigChangeMode = '🤷'
)

// DetectConfigChangeType encapsulates the fiddly logic for deciding what kind
// of Cloud configuration change we seem to be making, based on the existing
// working directory state (if any) and the current configuration.
//
// This is a pretty specialized sort of thing focused on finicky details of
// the way we currently model working directory settings and config, so its
// signature probably won't survive any non-trivial refactoring of how
// the CLI layer thinks about backends/state storage.
func DetectConfigChangeType(wdState *legacy.BackendState, config *configs.Backend, haveLocalStates bool) ConfigChangeMode {
	// Although externally the cloud integration isn't really a "backend",
	// internally we treat it a bit like one just to preserve all of our
	// existing interfaces that assume backends. "cloud" is the placeholder
	// name we use for it, even though that isn't a backend that's actually
	// available for selection in the usual way.
	wdIsCloud := wdState != nil && wdState.Type == "cloud"
	configIsCloud := config != nil && config.Type == "cloud"

	// "uninit" here means that the working directory is totally uninitialized,
	// even taking into account the possibility of implied local state that
	// therefore doesn't typically require explicit "terraform init".
	wdIsUninit := wdState == nil && !haveLocalStates

	switch {
	case configIsCloud:
		switch {
		case wdIsCloud || wdIsUninit:
			// If config has cloud and the working directory is completely
			// uninitialized then we assume we're doing the initial activation
			// of this working directory for an already-migrated-to-cloud
			// remote state.
			return ConfigChangeInPlace
		default:
			// Otherwise, we seem to be migrating into cloud mode from a backend.
			return ConfigMigrationIn
		}
	default:
		switch {
		case wdIsCloud:
			// If working directory is already cloud but config isn't, we're
			// migrating away from cloud to a backend.
			return ConfigMigrationOut
		default:
			// Otherwise, this situation seems to be something unrelated to
			// cloud mode and so outside of our scope here.
			return ConfigChangeIrrelevant
		}
	}

}

func (m ConfigChangeMode) InvolvesCloud() bool {
	switch m {
	case ConfigMigrationIn, ConfigMigrationOut, ConfigChangeInPlace:
		return true
	default:
		return false
	}
}

func (m ConfigChangeMode) IsCloudMigration() bool {
	switch m {
	case ConfigMigrationIn, ConfigMigrationOut:
		return true
	default:
		return false
	}
}
