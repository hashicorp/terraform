package e2etest

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/e2e"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	simple "github.com/hashicorp/terraform/internal/provider-simple"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	tfplugin "github.com/hashicorp/terraform/plugin"
)

// The tests in this file are for the "unmanaged provider workflow", which
// includes variants of the following sequence, with different details:
// terraform init
// terraform plan
// terraform apply
//
// These tests are run against an in-process server, and checked to make sure
// they're not trying to control the lifecycle of the binary. They are not
// checked for correctness of the operations themselves.

type reattachConfig struct {
	Protocol string
	Pid      int
	Test     bool
	Addr     reattachConfigAddr
}

type reattachConfigAddr struct {
	Network string
	String  string
}

type providerServer struct {
	sync.Mutex
	proto.ProviderServer
	planResourceChangeCalled  bool
	applyResourceChangeCalled bool
}

func (p *providerServer) PlanResourceChange(ctx context.Context, req *proto.PlanResourceChange_Request) (*proto.PlanResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.planResourceChangeCalled = true
	return p.ProviderServer.PlanResourceChange(ctx, req)
}

func (p *providerServer) ApplyResourceChange(ctx context.Context, req *proto.ApplyResourceChange_Request) (*proto.ApplyResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = true
	return p.ProviderServer.ApplyResourceChange(ctx, req)
}

func (p *providerServer) PlanResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.planResourceChangeCalled
}
func (p *providerServer) ResetPlanResourceChangeCalled() {
	p.Lock()
	defer p.Unlock()

	p.planResourceChangeCalled = false
}

func (p *providerServer) ApplyResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.applyResourceChangeCalled
}
func (p *providerServer) ResetApplyResourceChangeCalled() {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = false
}

func TestUnmanagedSeparatePlan(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "test-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})
	provider := &providerServer{
		ProviderServer: grpcwrap.Provider(simple.Provider()),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go plugin.Serve(&plugin.ServeConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugintest",
			Level:  hclog.Trace,
			Output: ioutil.Discard,
		}),
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachCh,
			CloseCh:          closeCh,
		},
		GRPCServer: plugin.DefaultGRPCServer,
		VersionedPlugins: map[int]plugin.PluginSet{
			5: plugin.PluginSet{
				"provider": &tfplugin.GRPCProviderPlugin{
					GRPCProvider: func() proto.ProviderServer {
						return provider
					},
				},
			},
		},
	})
	config := <-reattachCh
	if config == nil {
		t.Fatalf("no reattach config received")
	}
	reattachStr, err := json.Marshal(map[string]reattachConfig{
		"hashicorp/test": reattachConfig{
			Protocol: string(config.Protocol),
			Pid:      config.Pid,
			Test:     true,
			Addr: reattachConfigAddr{
				Network: config.Addr.Network(),
				String:  config.Addr.String(),
			},
		},
	})

	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))
	tf.AddEnv("PLUGIN_PROTOCOL_VERSION=5")

	//// INIT
	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	// Make sure we didn't download the binary
	if strings.Contains(stdout, "Installing hashicorp/test v") {
		t.Errorf("test provider download message is present in init output:\n%s", stdout)
	}
	if tf.FileExists(filepath.Join(".terraform", "plugins", "registry.terraform.io", "hashicorp", "test")) {
		t.Errorf("test provider binary found in .terraform dir")
	}

	//// PLAN
	_, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if !provider.PlanResourceChangeCalled() {
		t.Error("PlanResourceChange not called on un-managed provider")
	}

	//// APPLY
	_, stderr, err = tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !provider.ApplyResourceChangeCalled() {
		t.Error("ApplyResourceChange not called on un-managed provider")
	}
	provider.ResetApplyResourceChangeCalled()

	//// DESTROY
	_, stderr, err = tf.Run("destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected destroy error: %s\nstderr:\n%s", err, stderr)
	}

	if !provider.ApplyResourceChangeCalled() {
		t.Error("ApplyResourceChange (destroy) not called on in-process provider")
	}
	cancel()
	<-closeCh
}
