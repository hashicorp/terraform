// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

var (
	_ proto.CallbackServiceServer = (*Server)(nil)
)

// tracer is resolved lazily so the global TracerProvider installed by
// openTelemetryInit (which runs after package init) is reflected.
func tracer() trace.Tracer {
	return otel.Tracer("github.com/hashicorp/terraform/internal/policy/callback")
}

type Server struct {
	ID       uint32
	Registry Registry
	Grpc     *grpc.Server
	proto.UnimplementedCallbackServiceServer
}

func (s *Server) GetResources(ctx context.Context, request *proto.GetResourcesRequest) (*proto.GetResourcesResponse, error) {
	ctx, span := tracer().Start(ctx, "policy.callback.server.get_resources",
		trace.WithAttributes(
			attribute.String("policy.callback.resource_type", request.Type),
			attribute.Int64("policy.evaluation.id", int64(request.EvaluationRequestId)),
		),
	)
	defer span.End()

	attrs, err := msgpack.Unmarshal(request.Attributes, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to unserialize attributes: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		err := fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	resources, isPartialResult, err := functions.GetResources(ctx, request.Type, attrs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	results := make([][]byte, 0, len(resources))
	for _, resource := range resources {
		result, err := msgpack.Marshal(resource, cty.DynamicPseudoType)
		if err != nil {
			err = fmt.Errorf("failed to serialize resource: %w", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		results = append(results, result)
	}

	span.SetAttributes(attribute.Int("policy.callback.results.count", len(results)))
	return &proto.GetResourcesResponse{
		Results: results,
		Partial: isPartialResult,
	}, nil
}

func (s *Server) GetDataSource(ctx context.Context, request *proto.GetDataSourceRequest) (*proto.GetDataSourceResponse, error) {
	ctx, span := tracer().Start(ctx, "policy.callback.server.get_datasource",
		trace.WithAttributes(
			attribute.String("policy.callback.datasource_type", request.Type),
			attribute.Int64("policy.evaluation.id", int64(request.EvaluationRequestId)),
		),
	)
	defer span.End()

	config, err := msgpack.Unmarshal(request.Config, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to unserialize config: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		err := fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	datasource, err := functions.GetDataSource(ctx, request.Type, config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	result, err := msgpack.Marshal(datasource, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to serialize datasource: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return &proto.GetDataSourceResponse{
		Result: result,
	}, nil
}

func (s *Server) Stop() {
	if s.Grpc != nil {
		s.Grpc.GracefulStop()
	}
}
