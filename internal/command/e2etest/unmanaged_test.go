// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/e2e"
	proto5 "github.com/hashicorp/terraform/internal/tfplugin5"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
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
	Protocol        string
	ProtocolVersion int
	Pid             int
	Test            bool
	Addr            reattachConfigAddr
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
	listResourceCalled        bool
	readStateBytesCalled      bool
	writeStateBytesCalled     bool
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

func (p *providerServer) WriteStateBytes(server proto.Provider_WriteStateBytesServer) error {
	p.Lock()
	defer p.Unlock()

	p.writeStateBytesCalled = true
	return p.ProviderServer.WriteStateBytes(server)
}

func (p *providerServer) ReadStateBytes(req *proto.ReadStateBytes_Request, server proto.Provider_ReadStateBytesServer) error {
	p.Lock()
	defer p.Unlock()

	p.readStateBytesCalled = true
	return p.ProviderServer.ReadStateBytes(req, server)
}

func (p *providerServer) ListResource(req *proto.ListResource_Request, res proto.Provider_ListResourceServer) error {
	p.Lock()
	defer p.Unlock()

	p.listResourceCalled = true
	return p.ProviderServer.ListResource(req, res)
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

func (p *providerServer) ListResourceCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.listResourceCalled
}

func (p *providerServer) ReadStateBytesCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.readStateBytesCalled
}

func (p *providerServer) ResetReadStateBytesCalled() {
	p.Lock()
	defer p.Unlock()

	p.readStateBytesCalled = false
}

func (p *providerServer) WriteStateBytesCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.writeStateBytesCalled
}

func (p *providerServer) ResetWriteStateBytesCalled() {
	p.Lock()
	defer p.Unlock()

	p.writeStateBytesCalled = false
}

type providerServer5 struct {
	sync.Mutex
	proto5.ProviderServer
	planResourceChangeCalled  bool
	applyResourceChangeCalled bool
	listResourceCalled        bool
}

func (p *providerServer5) PlanResourceChange(ctx context.Context, req *proto5.PlanResourceChange_Request) (*proto5.PlanResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.planResourceChangeCalled = true
	return p.ProviderServer.PlanResourceChange(ctx, req)
}

func (p *providerServer5) ApplyResourceChange(ctx context.Context, req *proto5.ApplyResourceChange_Request) (*proto5.ApplyResourceChange_Response, error) {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = true

	return p.ProviderServer.ApplyResourceChange(ctx, req)
}

func (p *providerServer5) ListResource(req *proto5.ListResource_Request, res proto5.Provider_ListResourceServer) error {
	p.Lock()
	defer p.Unlock()

	p.listResourceCalled = true
	return p.ProviderServer.ListResource(req, res)
}

func (p *providerServer5) PlanResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.planResourceChangeCalled
}

func (p *providerServer5) ResetPlanResourceChangeCalled() {
	p.Lock()
	defer p.Unlock()

	p.planResourceChangeCalled = false
}

func (p *providerServer5) ApplyResourceChangeCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.applyResourceChangeCalled
}

func (p *providerServer5) ResetApplyResourceChangeCalled() {
	p.Lock()
	defer p.Unlock()

	p.applyResourceChangeCalled = false
}

func (p *providerServer5) ListResourceCalled() bool {
	p.Lock()
	defer p.Unlock()

	return p.listResourceCalled
}

func TestUnmanagedSeparatePlan(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "test-provider")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	reattachStr, provider := reattachedProviderForTest(t, addrs.NewDefaultProvider("test"), 6)
	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

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
}

func TestUnmanagedSeparatePlan_proto5(t *testing.T) {
	t.Parallel()

	fixturePath := filepath.Join("testdata", "test-provider")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	reattachStr, provider := reattachedProviderForTest(t, addrs.NewDefaultProvider("test"), 5) // protocol 5
	tf.AddEnv("TF_REATTACH_PROVIDERS=" + string(reattachStr))

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
}
