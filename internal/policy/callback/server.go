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
	// Objects encodes the resources that GetResources returns, and serves
	// GetResourcesByIdentifier. The policy client sets it.
	Objects Objects
	proto.UnimplementedCallbackServiceServer
}

// Objects encodes resource objects for the plugin, and resolves the
// identifiers it hands out.
type Objects interface {
	GetResourcesResponse(ctx context.Context, resources []Resource, partial bool) (*proto.GetResourcesResponse, error)
	GetResourcesByIdentifier(ctx context.Context, identifiers []string) *proto.GetResourcesByIdentifierResponse
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
	return s.Objects.GetResourcesResponse(ctx, resources, isPartialResult)
}

func (s *Server) GetResourcesByIdentifier(ctx context.Context, request *proto.GetResourcesByIdentifierRequest) (*proto.GetResourcesByIdentifierResponse, error) {
	return s.Objects.GetResourcesByIdentifier(ctx, request.Identifiers), nil
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
