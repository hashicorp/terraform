// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/version"
)

const (
	TerraformPolicyPluginEnvVar   = "TF_POLICY_PLUGIN"
	TerraformPolicyLogLevelEnvVar = "TF_POLICY_LOG_LEVEL"
	callbackServiceTimeout        = 10 * time.Second
)

var _ CallbackService = (*client)(nil)
var _ Client = (*client)(nil)

// pluginCrashDiagMsg is the sentinel detail fragment appended to diagnostics
// when a gRPC transport-level error indicates that the policy plugin process
// has crashed (segfault, OOM kill, broken pipe, etc.). Tests use this constant
// to assert the exact diagnostic content without coupling to the full message.
const pluginCrashDiagMsg = "The policy plugin process may have crashed or been forcibly terminated. Policy evaluation cannot be completed for this resource."

// isPluginCrashError reports whether err is a gRPC transport-level failure
// that is consistent with the plugin process having crashed or been killed.
// It recognises:
//   - io.EOF (connection closed by the server)
//   - gRPC Unavailable status (transport failure, broken pipe, connection reset)
//   - gRPC Internal status that carries connection-loss keywords
//   - gRPC Unknown status that carries EOF or connection-reset keywords
//
// It intentionally does NOT match DeadlineExceeded or Canceled, which have
// separate handling paths in the caller (per-call timeout and parent
// cancellation respectively).
func isPluginCrashError(err error) bool {
	if err == nil {
		return false
	}
	// Bare io.EOF: the plugin closed the connection without sending a response.
	if err == io.EOF {
		return true
	}
	st, ok := grpcstatus.FromError(err)
	if !ok {
		// Not a gRPC status error. Check for raw string matches that appear
		// when the transport layer wraps non-gRPC errors (e.g. net.OpError
		// with "broken pipe" or "connection reset by peer").
		msg := strings.ToLower(err.Error())
		return strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset") ||
			strings.Contains(msg, "eof")
	}
	switch st.Code() {
	case grpccodes.Unavailable:
		// Unavailable is the canonical gRPC code for transport failures:
		// the server process died, the connection was reset, etc.
		return true
	case grpccodes.Internal:
		// Internal can carry "EOF", "unexpected eof", or "broken pipe" when
		// the transport closes mid-message. msg is already lowercased so the
		// "eof" check subsumes "unexpected eof".
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "eof") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset")
	case grpccodes.Unknown:
		// Unknown wraps raw errors from the transport layer; check for the
		// same keywords.
		msg := strings.ToLower(st.Message())
		return strings.Contains(msg, "eof") ||
			strings.Contains(msg, "broken pipe") ||
			strings.Contains(msg, "connection reset")
	}
	return false
}

// transportErrorDiags builds the []*proto.Diagnostic slice for a transport-
// level plugin crash. It is separated from isPluginCrashError so that the
// detection and the message construction can be tested independently.
func transportErrorDiags(resourceLabel string, err error) []*proto.Diagnostic {
	return []*proto.Diagnostic{{
		Severity: proto.Severity_ERROR,
		Summary:  "Policy plugin crashed",
		Detail: fmt.Sprintf(
			"The policy plugin process crashed or became unavailable while evaluating %s: %v. %s",
			resourceLabel, err, pluginCrashDiagMsg,
		),
	}}
}

// NewPolicyClient initializes and connects to a new tfpolicy-plugin process
func NewPolicyClient(ctx context.Context, policyPluginPath string, policyPaths []string, ent *Entitlement) (Client, Diagnostics) {
	var diags Diagnostics
	client, err := Connect(ctx, policyPluginPath)
	if err != nil {
		diags = append(diags, NewErrorDiagnostic(
			"Failed to connect to policy engine",
			fmt.Sprintf("Failed to connect to policy engine: %s.", err),
			SetupErrorResult,
		))
		return nil, diags
	}

	var callbackServiceID uint32

	// initialize the callback service if the client supports it
	if srv, ok := client.(CallbackService); ok {
		callbackServer, cbDiags := srv.RegisterCallbackService(ctx)
		if cbDiags != nil {
			client.Stop()
			return nil, cbDiags
		}
		callbackServiceID = callbackServer.ID
	}

	resp := client.Setup(ctx, SetupRequest{
		SourceLocations: policyPaths,
		CallbackService: callbackServiceID,
		Entitlement:     ent,
	})
	diags = append(diags, resp.Diagnostics...)
	if diags.HasErrors() {
		client.Stop()
		return nil, diags
	}

	var requiredVersions constraints.IntersectionSpec
	for _, config := range resp.ServerConfigurations() {
		version, err := constraints.ParseRubyStyleMulti(config.RequiredVersion)
		if err != nil {
			diags = append(diags, NewErrorDiagnostic(
				"Failed to validate required Terraform version",
				fmt.Sprintf("The policy file %s had a Terraform version constraint that could not be parsed: %s.", config.File, err),
				SetupErrorResult,
			))
			continue
		}

		requiredVersions = append(requiredVersions, version...)
	}

	if diags.HasErrors() {
		client.Stop()
		return nil, diags
	}

	terraformVersion, err := versions.ParseVersion(version.Version)
	if err != nil {
		client.Stop()
		// This is crazy, it means the internal version number is invalid.
		panic(err)
	}

	constraint := versions.MeetingConstraints(requiredVersions)
	if !constraint.Has(terraformVersion) {
		diags = append(diags, NewErrorDiagnostic(
			"Invalid Terraform version for policies",
			fmt.Sprintf("The current version of Terraform is %s, and it is not compatible with the versions of Terraform required by the selected policies.", version.String()),
			SetupErrorResult,
		))
		client.Stop()
		return nil, diags
	}

	return client, diags
}

// Connect creates a connection to tfpolicy-plugin. If policyPluginPath is empty, the command lookup
// will default to the executable "tfpolicy-plugin" in the $PATH.
func Connect(ctx context.Context, policyPluginPath string) (Client, error) {
	ctx, span := tracer().Start(ctx, "policy.client.connect")
	defer span.End()

	pgm := "tfpolicy-plugin" // by default, just use this if it's in the path

	if policyPluginPath != "" {
		pgm = policyPluginPath
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
		Cmd:      cmd,
		AutoMTLS: true,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolGRPC,
		},
		// This propagates the active trace context via gRPC metadata  so the plugin's handler
		// spans become children of the terraform-side span that invoked them.
		GRPCDialOptions: []grpc.DialOption{
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
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
		err = fmt.Errorf("failed to connect to plugin: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	raw, err := rpc.Dispense("policy")
	if err != nil {
		plugin.Kill()
		err = fmt.Errorf("failed to dispense plugin: %v", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	sc := raw.(*client)
	sc.plugin = plugin
	return sc, nil
}

type client struct {
	plugin *plugin.Client

	broker           *plugin.GRPCBroker
	client           proto.PolicyClient
	callbackRegistry callback.Registry
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
		// Install the OpenTelemetry gRPC server stats handler so that the
		// trace context the plugin client injects into outgoing callback RPC
		// metadata is extracted and made the parent of the callback handler
		// spans.
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
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
	ctx, span := tracer().Start(ctx, "policy.client.setup")
	defer span.End()

	log.Printf("[DEBUG] Setting up Terraform Policy connection")
	protoReq := &proto.PolicySetupRequest{
		ClientCapabilities: new(proto.PolicySetupRequest_ClientCapabilities),
		SourceLocations:    req.SourceLocations,
		CallbackService:    req.CallbackService,
	}
	if req.Entitlement != nil {
		protoReq.Entitlement = &proto.PolicySetupRequest_Entitlement{
			Host:  req.Entitlement.Host,
			Token: req.Entitlement.Token,
			Org:   req.Entitlement.Org,
		}
	}
	response, err := c.client.Setup(ctx, protoReq)
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

func (c *client) EvaluateResource(ctx context.Context, req EvaluationRequest[*proto.PolicyEvaluateResourceRequest_ResourceMetadata]) EvaluationResponse {
	ctx, span := tracer().Start(ctx, "policy.client.evaluate_resource",
		trace.WithAttributes(attribute.String("policy.resource.type", req.Target)),
	)
	defer span.End()

	log.Printf("[DEBUG] Evaluating policy for resource %s", req.Target)
	var diags []*proto.Diagnostic

	req = normalizeRequest(req)

	attrs, err := resourceAttributesToProto(req.Attrs)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize attributes",
			Detail:   fmt.Sprintf("Failed to serialize attributes: %v.", err),
		}))
	}

	priorAttrs, err := resourceAttributesToProto(req.PriorAttrs)
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
		Attrs:        attrs,
		PriorAttrs:   priorAttrs,
		Metadata:     req.Meta,
	}

	// Register the callback functions with the callback service, so that they are available
	// for use during evaluation.
	c.callbackRegistry.Register(evalID, req.Callbacks)

	// We can unregister the callback functions after the evaluation is complete.
	defer c.callbackRegistry.Unregister(evalID)

	response, err := c.client.EvaluateResource(ctx, request)
	if err != nil {
		if isPluginCrashError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "plugin crash")
			log.Printf("[ERROR] policy plugin crashed during resource evaluation (%s): %v", req.Target, err)
			return ErrorEvalFromDiags(transportErrorDiags(req.Target, err))
		}
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) EvaluateProvider(ctx context.Context, req EvaluationRequest[*proto.PolicyEvaluateProviderRequest_ProviderMetadata]) EvaluationResponse {
	ctx, span := tracer().Start(ctx, "policy.client.evaluate_provider",
		trace.WithAttributes(attribute.String("policy.provider.type", req.Target)),
	)
	defer span.End()

	log.Printf("[DEBUG] Evaluating policy for provider %s", req.Target)
	var diags []*proto.Diagnostic
	req = normalizeRequest(req)

	attrs, err := resourceAttributesToProto(req.Attrs)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize attributes",
			Detail:   fmt.Sprintf("Failed to serialize attributes: %v.", err),
		}))
	}

	request := &proto.PolicyEvaluateProviderRequest{
		ProviderType: req.Target,
		Attrs:        attrs,
		Metadata:     req.Meta,
	}

	response, err := c.client.EvaluateProvider(ctx, request)
	if err != nil {
		if isPluginCrashError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "plugin crash")
			log.Printf("[ERROR] policy plugin crashed during provider evaluation (%s): %v", req.Target, err)
			return ErrorEvalFromDiags(transportErrorDiags(req.Target, err))
		}
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) EvaluateModule(ctx context.Context, req EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]) EvaluationResponse {
	ctx, span := tracer().Start(ctx, "policy.client.evaluate_module",
		trace.WithAttributes(attribute.String("policy.module.source", req.Target)),
	)
	defer span.End()

	log.Printf("[DEBUG] Evaluating policy for module %s", req.Target)
	var diags []*proto.Diagnostic

	req = normalizeRequest(req)

	attrs, err := resourceAttributesToProto(req.Attrs)
	if err != nil {
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to serialize attributes",
			Detail:   fmt.Sprintf("Failed to serialize attributes: %v.", err),
		}))
	}

	request := &proto.PolicyEvaluateModuleRequest{
		ModuleSource: req.Target,
		Attrs:        attrs,
		Metadata:     req.Meta,
	}

	response, err := c.client.EvaluateModule(ctx, request)
	if err != nil {
		if isPluginCrashError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "plugin crash")
			log.Printf("[ERROR] policy plugin crashed during module evaluation (%s): %v", req.Target, err)
			return ErrorEvalFromDiags(transportErrorDiags(req.Target, err))
		}
		return ErrorEvalFromDiags(append(diags, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Failed to evaluate Terraform Policy",
			Detail:   fmt.Sprintf("Failed to evaluate Terraform Policy: %v.", err),
		}))
	}

	return EvaluationFromProtoResponse(response.Result, response.PolicyDetails)
}

func (c *client) Stop() {
	log.Println("[DEBUG] stopping policy client")
	if c.cbServer != nil {
		c.cbServer.Stop()
	}
	c.plugin.Kill()
}

func normalizeRequest[T any](req EvaluationRequest[T]) EvaluationRequest[T] {
	attrs := req.Attrs
	priorAttrs := req.PriorAttrs
	if attrs.Raw == cty.NilVal {
		attrs.Raw = cty.EmptyObjectVal
	}
	if priorAttrs.Raw == cty.NilVal {
		priorAttrs.Raw = cty.EmptyObjectVal
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
