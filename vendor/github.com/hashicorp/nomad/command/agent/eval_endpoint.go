package agent

import (
	"net/http"
	"strings"

	"github.com/hashicorp/nomad/nomad/structs"
)

func (s *HTTPServer) EvalsRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.EvalListRequest{}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.EvalListResponse
	if err := s.agent.RPC("Eval.List", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Evaluations == nil {
		out.Evaluations = make([]*structs.Evaluation, 0)
	}
	return out.Evaluations, nil
}

func (s *HTTPServer) EvalSpecificRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	path := strings.TrimPrefix(req.URL.Path, "/v1/evaluation/")
	switch {
	case strings.HasSuffix(path, "/allocations"):
		evalID := strings.TrimSuffix(path, "/allocations")
		return s.evalAllocations(resp, req, evalID)
	default:
		return s.evalQuery(resp, req, path)
	}
}

func (s *HTTPServer) evalAllocations(resp http.ResponseWriter, req *http.Request, evalID string) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.EvalSpecificRequest{
		EvalID: evalID,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.EvalAllocationsResponse
	if err := s.agent.RPC("Eval.Allocations", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Allocations == nil {
		out.Allocations = make([]*structs.AllocListStub, 0)
	}
	return out.Allocations, nil
}

func (s *HTTPServer) evalQuery(resp http.ResponseWriter, req *http.Request, evalID string) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	args := structs.EvalSpecificRequest{
		EvalID: evalID,
	}
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var out structs.SingleEvalResponse
	if err := s.agent.RPC("Eval.GetEval", &args, &out); err != nil {
		return nil, err
	}

	setMeta(resp, &out.QueryMeta)
	if out.Eval == nil {
		return nil, CodedError(404, "eval not found")
	}
	return out.Eval, nil
}
