package nomad

import "github.com/hashicorp/nomad/nomad/structs"

// Region is used to query and list the known regions
type Region struct {
	srv *Server
}

// List is used to list all of the known regions. No leader forwarding is
// required for this endpoint because memberlist is used to populate the
// peers list we read from.
func (r *Region) List(args *structs.GenericRequest, reply *[]string) error {
	*reply = r.srv.Regions()
	return nil
}
