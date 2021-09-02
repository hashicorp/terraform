package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	fileprovisioner "github.com/hashicorp/terraform/internal/builtin/provisioners/file"
	localexec "github.com/hashicorp/terraform/internal/builtin/provisioners/local-exec"
	remoteexec "github.com/hashicorp/terraform/internal/builtin/provisioners/remote-exec"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"github.com/hashicorp/terraform/internal/command/plugins"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
)

// The TF_DISABLE_PLUGIN_TLS environment variable is intended only for use by
// the plugin SDK test framework, to reduce startup overhead when rapidly
// launching and killing lots of instances of the same provider.
//
// This is not intended to be set by end-users.
var enableProviderAutoMTLS = os.Getenv("TF_DISABLE_PLUGIN_TLS") == ""

// basePluginFinder creates an initial plugin finder that knows various
// settings that don't vary based on what command we're running, etc.
//
// The result will need further customization before it's useful to search
// for providers, which happens gradually as we work through the command
// package logic and then the backend.LocalRun logic.
//
// (This design where various different layers all contribute to the same
// object is unfortunate. It represents an intermediate step towards hopefully
// one day having this concern managed all in one place, but is better than
// what proceeded it where the encapsulated plugin detection logic _itself_ was
// spread over all of these different layers.)
func basePluginFinder(
	providerCacheDir string,
	providerDevOverrides map[addrs.Provider]getproviders.PackageLocalDir,
	unmanagedProviders map[addrs.Provider]*plugin.ReattachConfig,
) plugins.Finder {
	settings := plugins.FinderBaseSettings{
		ProviderDir:           providerCacheDir,
		ProvisionerSearchDirs: globalPluginDirs(),
		ProviderDevOverrides:  providerDevOverrides,
		UnmanagedProviders:    unmanagedProviders,
		BuiltinProviders:      builtinProviders(),
		BuiltinProvisioners:   builtinProvisioners(),
	}
	ret := plugins.NewFinder(settings)
	if !enableProviderAutoMTLS {
		ret = ret.WithoutProviderAutoMTLS()
	}
	return ret
}

// globalPluginDirs returns directories that should be searched for
// globally-installed plugins (not specific to the current configuration).
//
// Earlier entries in this slice get priority over later when multiple copies
// of the same plugin version are found, but newer versions always override
// older versions where both satisfy the provider version constraints.
func globalPluginDirs() []string {
	var ret []string
	// Look in ~/.terraform.d/plugins/ , or its equivalent on non-UNIX
	dir, err := cliconfig.ConfigDir()
	if err != nil {
		log.Printf("[ERROR] Error finding global config directory: %s", err)
	} else {
		machineDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		ret = append(ret, filepath.Join(dir, "plugins"))
		ret = append(ret, filepath.Join(dir, "plugins", machineDir))
	}

	return ret
}

// builtinProviders defines the providers that belong to the
// terraform.io/builtin/ namespace, which are compiled directly into
// Terraform CLI rather than being run as external processes.
func builtinProviders() map[string]providers.Factory {
	return map[string]providers.Factory{
		"terraform": func() (providers.Interface, error) {
			return terraformProvider.NewProvider(), nil
		},
		"test": func() (providers.Interface, error) {
			return moduletest.NewProvider(), nil
		},
	}
}

// builtinProvisioners defines the provisioners that are compiled directly
// into Terraform CLI, rather than being run as external processes.
//
// Unlike providers, provisioners have a flat namespace and so external
// plugins can potentially override the built-in provisioners, making the
// built-in functionality unreachable in that context.
func builtinProvisioners() map[string]provisioners.Factory {
	return map[string]provisioners.Factory{
		"file":        provisioners.FactoryFixed(fileprovisioner.New()),
		"local-exec":  provisioners.FactoryFixed(localexec.New()),
		"remote-exec": provisioners.FactoryFixed(remoteexec.New()),
	}
}
