package agent

import (
	"net/http"

	"github.com/hashicorp/nomad/nomad/structs"
)

func (s *HTTPServer) StatusLeaderRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	var args structs.GenericRequest
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var leader string
	if err := s.agent.RPC("Status.Leader", &args, &leader); err != nil {
		return nil, err
	}
	return leader, nil
}

func (s *HTTPServer) StatusPeersRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	var args structs.GenericRequest
	if s.parse(resp, req, &args.Region, &args.QueryOptions) {
		return nil, nil
	}

	var peers []string
	if err := s.agent.RPC("Status.Peers", &args, &peers); err != nil {
		return nil, err
	}
	if len(peers) == 0 {
		peers = make([]string, 0)
	}
	return peers, nil
}
