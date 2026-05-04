// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

const (
	TerraformPolicyPluginEnvVar   = "TF_POLICY_PLUGIN"
	TerraformPolicyLogLevelEnvVar = "TF_POLICY_LOG_LEVEL"
	callbackServiceTimeout        = 10 * time.Second
)

var _ CallbackService = (*client)(nil)
var _ Client = (*client)(nil)

func Connect(ctx context.Context) (Client, error) {

	pgm := "tfpolicy-plugin" // by default, just use this if it's in the path
	if envvar := os.Getenv(TerraformPolicyPluginEnvVar); len(envvar) > 0 {
		pgm = envvar
	}

	cmd := exec.CommandContext(ctx, pgm, "rpcapi")
	plugin := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "TF_POLICY_PLUGIN",
			MagicCookieValue: "6F11ED78A2AB",
		},
		Plugins: map[string]plugin.Plugin{
			"policy": new(policy),
		},
		Cmd: cmd,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		Logger: hclog.New(&hclog.LoggerOptions{
			Level: func() hclog.Level {
				level := hclog.LevelFromString(os.Getenv(TerraformPolicyLogLevelEnvVar))
				if level == hclog.NoLevel {
					return hclog.Error
				}
				return level
			}(),
		}),
	})

	rpc, err := plugin.Client()
	if err != nil {
		plugin.Kill()
		return nil, fmt.Errorf("failed to connect to plugin: %v", err)
	}

	raw, err := rpc.Dispense("policy")
	if err != nil {
		plugin.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %v", err)
	}

	sc := raw.(*client)
	sc.plugin = plugin
	return sc, nil
}

type client struct {
	plugin *plugin.Client

	broker           *plugin.GRPCBroker
	client           proto.PolicyClient
	callbackRegistry *callback.InternalRegistry
	cbServer         *callback.Server
}

func (c *client) RegisterCallbackService(ctx context.Context) (*callback.Server, Diagnostics) {
	if c.cbServer != nil {
		panic("callback service already registered")
	}

	cbServiceID := c.broker.NextId()
	c.cbServer = &callback.Server{
		ID:       cbServiceID,
		Registry: c.callbackRegistry,
	}

	serverCh := make(chan *grpc.Server, 1)

	// Start the callback service server on the broker.
	go c.broker.AcceptAndServe(cbServiceID, func(opts []grpc.ServerOption) *grpc.Server {
		server := grpc.NewServer(opts...)
		proto.RegisterCallbackServiceServer(server, c.cbServer)
		serverCh <- server
		return server
	})

	select {
	// Wait for the server to be ready before returning.
	case server := <-serverCh:
		c.cbServer.Grpc = server
	// If the context is done, return early with an error.
	case <-ctx.Done():
		return nil, Diagnostics{
			NewErrorDiagnostic("Failed to register callback service",
				fmt.Sprintf("Failed to register callback service: %v.", ctx.Err()),
				SetupErrorResult,
			),
		}

	// If the wait has exceeded the timeout, return early with an error.
	case <-time.After(callbackServiceTimeout):
		return nil, Diagnostics{
			NewErrorDiagnostic("Failed to register callback service",
				fmt.Sprintf("Failed to register callback service: timed out after %s.", callbackServiceTimeout),
				SetupErrorResult,
			),
		}
	}

	return c.cbServer, nil
}

func (c *client) Setup(ctx context.Context, req SetupRequest) SetupResponse {
	log.Printf("[DEBUG] Setting up Terraform Policy connection")
	response, err := c.client.Setup(ctx, &proto.PolicySetupRequest{
		ClientCapabilities: new(proto.PolicySetupRequest_ClientCapabilities),
		SourceLocations:    req.SourceLocations,
		CallbackService:    req.CallbackService,
	})
	if err != nil {
		return SetupResponse{Diagnostics: Diagnostics{
			NewErrorDiagnostic("Failed to setup Terraform Policy connection",
				fmt.Sprintf("Failed to setup Terraform Policy connection: %v.", err),
				SetupErrorResult,
			),
		}}
	}

	return SetupResponse{
		serverCapabilities: response.ServerCapabilities,
		Diagnostics:        DiagsFromProto(response.Diagnostics, nil),
	}
}

func (c *client) Evaluate(ctx context.Context, req EvaluationRequest[*proto.ResourceMetadata]) EvaluationResponse {
	log.Printf("[DEBUG] Evaluating policy for resource %s", req.Target)
	var diags []*proto.Diagnostic

	req = normalizeRequest(req)

	attrsBytes, err := msgpack.Marshal(req.Attrs, cty.DynamicPseudoType)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize attributes",
			Detail:   fmt.Sprintf("Failed to serialize attributes: %v.", err),
		}))
	}

	priorAttrsBytes, err := msgpack.Marshal(req.PriorAttrs, cty.DynamicPseudoType)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize prior attributes",
			Detail:   fmt.Sprintf("Failed to serialize prior attributes: %v.", err),
		}))
	}

	evalID := c.callbackRegistry.NextID()
	request := &proto.PolicyEvaluateResourceRequest{
		EvaluationId: evalID,
		Resource:     req.Target,
		Attrs:        attrsBytes,
		Metadata:     req.Meta,
		PriorAttrs:   priorAttrsBytes,
	}

	// Register the callback functions with the callback service, so that they are available
	// for use during evaluation.
	c.callbackRegistry.Register(evalID, req.Callbacks)

	// We can unregister the callback functions after the evaluation is complete.
	defer c.callbackRegistry.Unregister(evalID)

	response, err := c.client.EvaluateResource(ctx, request)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) EvaluateProvider(ctx context.Context, req EvaluationRequest[*proto.ProviderMetadata]) EvaluationResponse {
	log.Printf("[DEBUG] Evaluating policy for provider %s", req.Target)
	var diags []*proto.Diagnostic
	req = normalizeRequest(req)

	attrsBytes, err := msgpack.Marshal(req.Attrs, cty.DynamicPseudoType)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize attributes",
			Detail:   fmt.Sprintf("Failed to serialize attributes: %v.", err),
		}))
	}

	request := &proto.PolicyEvaluateProviderRequest{
		ProviderType: req.Target,
		Attrs:        attrsBytes,
		Metadata:     req.Meta,
	}

	response, err := c.client.EvaluateProvider(ctx, request)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) EvaluateModule(ctx context.Context, req EvaluationRequest[*proto.ModuleMetadata]) EvaluationResponse {
	log.Printf("[DEBUG] Evaluating policy for module %s", req.Target)
	var diags []*proto.Diagnostic

	req = normalizeRequest(req)

	request := &proto.PolicyEvaluateModuleRequest{
		ModuleSource: req.Target,
		Metadata:     req.Meta,
	}

	response, err := c.client.EvaluateModule(ctx, request)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) Stop() {
	if c.cbServer != nil {
		c.cbServer.Stop()
	}
	c.plugin.Kill()
}

func normalizeRequest[T any](req EvaluationRequest[T]) EvaluationRequest[T] {
	attrs := req.Attrs
	priorAttrs := req.PriorAttrs
	if attrs == cty.NilVal {
		attrs = cty.EmptyObjectVal
	}
	if priorAttrs == cty.NilVal {
		priorAttrs = cty.EmptyObjectVal
	}

	return EvaluationRequest[T]{
		Target:     req.Target,
		Attrs:      attrs,
		PriorAttrs: priorAttrs,
		Meta:       req.Meta,
		Callbacks:  req.Callbacks,
	}
}

type policy struct {
	plugin.NetRPCUnsupportedPlugin
}

func (s *policy) GRPCServer(*plugin.GRPCBroker, *grpc.Server) error {
	// This package is only implementing the client side of the Terraform Policy
	// plugin.
	return fmt.Errorf("server configuration not supported")
}

func (s *policy) GRPCClient(_ context.Context, broker *plugin.GRPCBroker, conn *grpc.ClientConn) (interface{}, error) {
	return &client{
		plugin:           nil, // this will be set by the Connect function
		broker:           broker,
		client:           proto.NewPolicyClient(conn),
		callbackRegistry: callback.NewRegistry(),
	}, nil
}
