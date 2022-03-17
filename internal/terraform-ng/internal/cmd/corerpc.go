package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/logging"
	tfplugin "github.com/hashicorp/terraform/internal/plugin"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/terraform"

	"github.com/spf13/cobra"
)

var coreRPCCmd = &cobra.Command{
	Use:   "core-rpc",
	Short: "Run as an RPC server exposing Terraform Core.",
	Long: `Runs a gRPC-based RPC server exposing Terraform Core functionality directly.
	
This is an internal plumbing command that most users should not need to use directly.`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()

		cacheDir, err := providerCacheDir()
		if err != nil {
			cmd.PrintErrf("Can't find provider cache dir: %s.\n", err)
			os.Exit(1)
		}

		providerFactories, err := hardcodedProviderFactories(cacheDir)
		if err != nil {
			cmd.PrintErrf("Can't prepare the hard-coded providers: %s.\n", err)
			os.Exit(1)
		}

		if !rpcapi.RunningAsPlugin(ctx) {
			cmd.PrintErrln("The core-rpc command is only for use by other wrapper programs that understand its RPC protocol.")
			os.Exit(1)
		}

		err = rpcapi.Serve(ctx, rpcapi.ServeOpts{
			GetCoreOpts: func() *terraform.ContextOpts {
				return &terraform.ContextOpts{
					Providers: providerFactories,
				}
			},
			WorkingDir: ".",
			ModulesDir: ".terraform/modules", // NOTE: Nothing is actually populating this, so in practice we cannot have child modules
		})
		if err != nil {
			cmd.PrintErrf("Failed to launch RPC server: %s\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(coreRPCCmd)
}

func providerCacheDir() (*providercache.Dir, error) {
	cacheDir := os.Getenv("TF_NG_TMP_PROVIDER_CACHE_DIR")
	if cacheDir == "" {
		return nil, fmt.Errorf("environment variable TF_NG_TMP_PROVIDER_CACHE_DIR isn't set")
	}

	return providercache.NewDir(cacheDir), nil
}

func hardcodedProviderFactories(dir *providercache.Dir) (map[addrs.Provider]providers.Factory, error) {
	availableProviders := []addrs.Provider{
		addrs.MustParseProviderSourceString("hashicorp/aws"),
		addrs.MustParseProviderSourceString("hashicorp/kubernetes"),
		addrs.MustParseProviderSourceString("hashicorp/google"),
	}

	ret := make(map[addrs.Provider]providers.Factory)
	for _, providerAddr := range availableProviders {
		cached := dir.ProviderLatestVersion(providerAddr)
		if cached == nil {
			return nil, fmt.Errorf("no plugin available for %s", providerAddr)
		}
		ret[providerAddr] = providerFactory(cached)
	}
	return ret, nil
}

// providerFactory produces a provider factory that runs up the executable
// file in the given cache package and uses go-plugin to implement
// providers.Interface against it.
func providerFactory(meta *providercache.CachedProvider) providers.Factory {
	return func() (providers.Interface, error) {
		execFile, err := meta.ExecutableFile()
		if err != nil {
			return nil, err
		}

		config := &plugin.ClientConfig{
			HandshakeConfig:  tfplugin.Handshake,
			Logger:           logging.NewProviderLogger(""),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Managed:          true,
			Cmd:              exec.Command(execFile),
			AutoMTLS:         true,
			VersionedPlugins: tfplugin.VersionedPlugins,
			SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", meta.Provider)),
			SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", meta.Provider)),
		}

		client := plugin.NewClient(config)
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		// store the client so that the plugin can kill the child process
		protoVer := client.NegotiatedVersion()
		switch protoVer {
		case 5:
			p := raw.(*tfplugin.GRPCProvider)
			p.PluginClient = client
			return p, nil
		case 6:
			p := raw.(*tfplugin6.GRPCProvider)
			p.PluginClient = client
			return p, nil
		default:
			panic("unsupported protocol version")
		}
	}
}
