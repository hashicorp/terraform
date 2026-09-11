// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"context"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

var (
	_ proto.CallbackServiceServer = (*Server)(nil)
)

type Server struct {
	ID       uint32
	Registry Registry
	Grpc     *grpc.Server
	proto.UnimplementedCallbackServiceServer
}

func (s *Server) GetResources(ctx context.Context, request *proto.GetResourcesRequest) (*proto.GetResourcesResponse, error) {
	attrs, err := msgpack.Unmarshal(request.Attributes, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to unserialize attributes: %w", err)
		return nil, err
	}
	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		err := fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
		return nil, err
	}
	resources, isPartialResult, err := functions.GetResources(ctx, request.Type, attrs)
	if err != nil {
		return nil, err
	}

	results := make([][]byte, 0, len(resources))
	for _, resource := range resources {
		result, err := msgpack.Marshal(resource, cty.DynamicPseudoType)
		if err != nil {
			err = fmt.Errorf("failed to serialize resource: %w", err)
			return nil, err
		}
		results = append(results, result)
	}

	return &proto.GetResourcesResponse{
		Results: results,
		Partial: isPartialResult,
	}, nil
}

func (s *Server) RelatedResources(ctx context.Context, request *proto.RelatedResourcesRequest) (*proto.RelatedResourcesResponse, error) {
	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		err := fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
		return nil, err
	}
	pairs := make([]RelatedAttributePair, 0, len(request.AttributePairs))
	for _, pair := range request.AttributePairs {
		pairs = append(pairs, RelatedAttributePair{
			SourceAttribute:  pair.SourceAttribute,
			RelatedAttribute: pair.RelatedAttribute,
		})
	}
	resources, isPartialResult, err := functions.RelatedResources(ctx, request.Type, pairs)
	if err != nil {
		return nil, err
	}

	results := make([][]byte, 0, len(resources))
	for _, resource := range resources {
		result, err := msgpack.Marshal(resource, cty.DynamicPseudoType)
		if err != nil {
			err = fmt.Errorf("failed to serialize resource: %w", err)
			return nil, err
		}
		results = append(results, result)
	}

	return &proto.RelatedResourcesResponse{
		Results: results,
		Partial: isPartialResult,
	}, nil
}

func (s *Server) GetDataSource(ctx context.Context, request *proto.GetDataSourceRequest) (*proto.GetDataSourceResponse, error) {
	config, err := msgpack.Unmarshal(request.Config, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to unserialize config: %w", err)

		return nil, err
	}

	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		err := fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
		return nil, err
	}
	datasource, isDeferred, err := functions.GetDataSource(ctx, request.Type, config)
	if err != nil {
		return nil, err
	}

	result, err := msgpack.Marshal(datasource, cty.DynamicPseudoType)
	if err != nil {
		err = fmt.Errorf("failed to serialize datasource: %w", err)
		return nil, err
	}

	return &proto.GetDataSourceResponse{
		Result:   result,
		Deferred: isDeferred,
	}, nil
}

func (s *Server) Stop() {
	if s.Grpc != nil {
		s.Grpc.GracefulStop()
	}
}
