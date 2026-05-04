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
	Registry *InternalRegistry
	Grpc     *grpc.Server
	proto.UnimplementedCallbackServiceServer
}

func (s *Server) GetResources(_ context.Context, request *proto.GetResourcesRequest) (*proto.GetResourcesResponse, error) {
	attrs, err := msgpack.Unmarshal(request.Data, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("failed to unserialize data: %w", err)
	}
	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		return nil, fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
	}
	resources, err := functions.GetResources(request.Type, attrs)
	if err != nil {
		return nil, err
	}

	results := make([][]byte, 0, len(resources))
	for _, resource := range resources {
		result, err := msgpack.Marshal(resource, cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize data: %w", err)
		}
		results = append(results, result)
	}

	return &proto.GetResourcesResponse{
		Results: results,
	}, nil
}

func (s *Server) GetDataSource(_ context.Context, request *proto.GetDataSourceRequest) (*proto.GetDataSourceResponse, error) {
	attrs, err := msgpack.Unmarshal(request.Data, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("failed to unserialize data: %w", err)
	}

	functions, ok := s.Registry.Get(request.EvaluationRequestId)
	if !ok {
		return nil, fmt.Errorf("no callback registered for ID %d (request type: %s)", request.EvaluationRequestId, request.Type)
	}
	datasource, err := functions.GetDataSource(request.Type, attrs)
	if err != nil {
		return nil, err
	}

	result, err := msgpack.Marshal(datasource, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
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
