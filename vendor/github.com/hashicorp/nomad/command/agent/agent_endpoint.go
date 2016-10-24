package agent

import (
	"net"
	"net/http"

	"github.com/hashicorp/serf/serf"
)

type Member struct {
	Name        string
	Addr        net.IP
	Port        uint16
	Tags        map[string]string
	Status      string
	ProtocolMin uint8
	ProtocolMax uint8
	ProtocolCur uint8
	DelegateMin uint8
	DelegateMax uint8
	DelegateCur uint8
}

func nomadMember(m serf.Member) Member {
	return Member{
		Name:        m.Name,
		Addr:        m.Addr,
		Port:        m.Port,
		Tags:        m.Tags,
		Status:      m.Status.String(),
		ProtocolMin: m.ProtocolMin,
		ProtocolMax: m.ProtocolMax,
		ProtocolCur: m.ProtocolCur,
		DelegateMin: m.DelegateMin,
		DelegateMax: m.DelegateMax,
		DelegateCur: m.DelegateCur,
	}
}

func (s *HTTPServer) AgentSelfRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}

	// Get the member as a server
	var member serf.Member
	srv := s.agent.Server()
	if srv != nil {
		member = srv.LocalMember()
	}

	self := agentSelf{
		Config: s.agent.config,
		Member: nomadMember(member),
		Stats:  s.agent.Stats(),
	}
	return self, nil
}

func (s *HTTPServer) AgentJoinRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "PUT" && req.Method != "POST" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	srv := s.agent.Server()
	if srv == nil {
		return nil, CodedError(501, ErrInvalidMethod)
	}

	// Get the join addresses
	query := req.URL.Query()
	addrs := query["address"]
	if len(addrs) == 0 {
		return nil, CodedError(400, "missing address to join")
	}

	// Attempt the join
	num, err := srv.Join(addrs)
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return joinResult{num, errStr}, nil
}

func (s *HTTPServer) AgentMembersRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	srv := s.agent.Server()
	if srv == nil {
		return nil, CodedError(501, ErrInvalidMethod)
	}

	serfMembers := srv.Members()
	members := make([]Member, len(serfMembers))
	for i, mem := range serfMembers {
		members[i] = nomadMember(mem)
	}
	return members, nil
}

func (s *HTTPServer) AgentForceLeaveRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "PUT" && req.Method != "POST" {
		return nil, CodedError(405, ErrInvalidMethod)
	}
	srv := s.agent.Server()
	if srv == nil {
		return nil, CodedError(501, ErrInvalidMethod)
	}

	// Get the node to eject
	node := req.URL.Query().Get("node")
	if node == "" {
		return nil, CodedError(400, "missing node to force leave")
	}

	// Attempt remove
	err := srv.RemoveFailedNode(node)
	return nil, err
}

// AgentServersRequest is used to query the list of servers used by the Nomad
// Client for RPCs.  This endpoint can also be used to update the list of
// servers for a given agent.
func (s *HTTPServer) AgentServersRequest(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	switch req.Method {
	case "PUT", "POST":
		return s.updateServers(resp, req)
	case "GET":
		return s.listServers(resp, req)
	default:
		return nil, CodedError(405, ErrInvalidMethod)
	}
}

func (s *HTTPServer) listServers(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	client := s.agent.Client()
	if client == nil {
		return nil, CodedError(501, ErrInvalidMethod)
	}

	peers := s.agent.client.RPCProxy().ServerRPCAddrs()
	return peers, nil
}

func (s *HTTPServer) updateServers(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	client := s.agent.Client()
	if client == nil {
		return nil, CodedError(501, ErrInvalidMethod)
	}

	// Get the servers from the request
	servers := req.URL.Query()["address"]
	if len(servers) == 0 {
		return nil, CodedError(400, "missing server address")
	}

	// Set the servers list into the client
	for _, server := range servers {
		s.agent.logger.Printf("[TRACE] Adding server %s to the client's primary server list", server)
		se := client.AddPrimaryServerToRPCProxy(server)
		if se == nil {
			s.agent.logger.Printf("[ERR] Attempt to add server %q to client failed", server)
		}
	}
	return nil, nil
}

type agentSelf struct {
	Config *Config                      `json:"config"`
	Member Member                       `json:"member,omitempty"`
	Stats  map[string]map[string]string `json:"stats"`
}

type joinResult struct {
	NumJoined int    `json:"num_joined"`
	Error     string `json:"error"`
}
