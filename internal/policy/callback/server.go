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
	// Values encodes the resources that GetResources returns, and serves
	// GetResourceValues. The policy client sets it.
	Values Values
	proto.UnimplementedCallbackServiceServer
}

// Values encodes resource values for the plugin, and resolves the value IDs
// it hands out.
type Values interface {
	GetResourcesResponse(ctx context.Context, resources []Resource, partial bool) (*proto.GetResourcesResponse, error)
	GetResourceValues(ctx context.Context, ids []string) ([]*proto.ResourceAttributes, error)
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
	return s.Values.GetResourcesResponse(ctx, resources, isPartialResult)
}

func (s *Server) GetResourceValues(ctx context.Context, request *proto.GetResourceValuesRequest) (*proto.GetResourceValuesResponse, error) {
	values, err := s.Values.GetResourceValues(ctx, request.Ids)
	if err != nil {
		return nil, err
	}
	return &proto.GetResourceValuesResponse{Values: values}, nil
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
