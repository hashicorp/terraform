package aws

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
)

func resource_aws_route_table_create(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Create the routing table
	createOpts := &ec2.CreateRouteTable{
		VpcId: d.Attributes["vpc_id"].New,
	}
	log.Printf("[DEBUG] RouteTable create config: %#v", createOpts)
	resp, err := ec2conn.CreateRouteTable(createOpts)
	if err != nil {
		return nil, fmt.Errorf("Error creating route table: %s", err)
	}

	// Get the ID and store it
	rt := &resp.RouteTable
	s.ID = rt.RouteTableId
	log.Printf("[INFO] Route Table ID: %s", s.ID)

	// Wait for the route table to become available
	log.Printf(
		"[DEBUG] Waiting for route table (%s) to become available",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "ready",
		Refresh: RouteTableStateRefreshFunc(ec2conn, s.ID),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return s, fmt.Errorf(
			"Error waiting for route table (%s) to become available: %s",
			s.ID, err)
	}

	// Update our routes
	return resource_aws_route_table_update(s, d, meta)
}

func resource_aws_route_table_update(
	s *terraform.ResourceState,
	d *terraform.ResourceDiff,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// Our resulting state
	rs := s.MergeDiff(d)

	// Get our routes out of the merge
	oldroutes := flatmap.Expand(s.Attributes, "route")
	routes := flatmap.Expand(s.MergeDiff(d).Attributes, "route")

	// Determine the route operations we need to perform
	ops := routeTableOps(oldroutes, routes)
	if len(ops) == 0 {
		return s, nil
	}

	// Go through each operation, performing each one at a time.
	// We store the updated state on each operation so that if any
	// individual operation fails, we can return a valid partial state.
	var err error
	resultRoutes := make([]map[string]string, 0, len(ops))
	for _, op := range ops {
		switch op.Op {
		case routeTableOpCreate:
			opts := ec2.CreateRoute{
				RouteTableId:         s.ID,
				DestinationCidrBlock: op.Route.DestinationCidrBlock,
				GatewayId:            op.Route.GatewayId,
				InstanceId:           op.Route.InstanceId,
			}

			_, err = ec2conn.CreateRoute(&opts)
		case routeTableOpReplace:
			opts := ec2.ReplaceRoute{
				RouteTableId:         s.ID,
				DestinationCidrBlock: op.Route.DestinationCidrBlock,
				GatewayId:            op.Route.GatewayId,
				InstanceId:           op.Route.InstanceId,
			}

			_, err = ec2conn.ReplaceRoute(&opts)
		case routeTableOpDelete:
			_, err = ec2conn.DeleteRoute(
				s.ID, op.Route.DestinationCidrBlock)
		}

		if err != nil {
			// Exit early so we can return what we've done so far
			break
		}

		// If we didn't delete the route, append it to the list of routes
		// we have.
		if op.Op != routeTableOpDelete {
			resultMap := map[string]string{"cidr_block": op.Route.DestinationCidrBlock}
			if op.Route.GatewayId != "" {
				resultMap["gateway_id"] = op.Route.GatewayId
			} else if op.Route.InstanceId != "" {
				resultMap["instance_id"] = op.Route.InstanceId
			}

			resultRoutes = append(resultRoutes, resultMap)
		}
	}

	// Update our state with the settings
	flatmap.Map(rs.Attributes).Merge(flatmap.Flatten(map[string]interface{}{
		"route": resultRoutes,
	}))

	return rs, err
}

func resource_aws_route_table_destroy(
	s *terraform.ResourceState,
	meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// First request the routing table since we'll have to disassociate
	// all the subnets first.
	rtRaw, _, err := RouteTableStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		return nil
	}
	rt := rtRaw.(*ec2.RouteTable)

	// Do all the disassociations
	for _, a := range rt.Associations {
		log.Printf("[INFO] Disassociating association: %s", a.AssociationId)
		if _, err := ec2conn.DisassociateRouteTable(a.AssociationId); err != nil {
			return err
		}
	}

	// Delete the route table
	log.Printf("[INFO] Deleting Route Table: %s", s.ID)
	if _, err := ec2conn.DeleteRouteTable(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table: %s", err)
	}

	// Wait for the route table to really destroy
	log.Printf(
		"[DEBUG] Waiting for route table (%s) to become destroyed",
		s.ID)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"ready"},
		Target:  "",
		Refresh: RouteTableStateRefreshFunc(ec2conn, s.ID),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for route table (%s) to become destroyed: %s",
			s.ID, err)
	}

	return nil
}

func resource_aws_route_table_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	rtRaw, _, err := RouteTableStateRefreshFunc(ec2conn, s.ID)()
	if err != nil {
		return s, err
	}
	if rtRaw == nil {
		return nil, nil
	}

	rt := rtRaw.(*ec2.RouteTable)
	return resource_aws_route_table_update_state(s, rt)
}

func resource_aws_route_table_diff(
	s *terraform.ResourceState,
	c *terraform.ResourceConfig,
	meta interface{}) (*terraform.ResourceDiff, error) {
	b := &diff.ResourceBuilder{
		Attrs: map[string]diff.AttrType{
			"vpc_id": diff.AttrTypeCreate,
			"route":  diff.AttrTypeUpdate,
		},
	}

	return b.Diff(s, c)
}

func resource_aws_route_table_update_state(
	s *terraform.ResourceState,
	rt *ec2.RouteTable) (*terraform.ResourceState, error) {
	s.Attributes["vpc_id"] = rt.VpcId

	// We belong to a VPC
	s.Dependencies = []terraform.ResourceDependency{
		terraform.ResourceDependency{ID: rt.VpcId},
	}

	return s, nil
}

// routeTableOp represents a minor operation on the routing table.
// This tells us what we should do to the routing table.
type routeTableOp struct {
	Op    routeTableOpType
	Route ec2.Route
}

// routeTableOpType is the type of operation related to a route that
// can be operated on a routing table.
type routeTableOpType byte

const (
	routeTableOpCreate routeTableOpType = iota
	routeTableOpReplace
	routeTableOpDelete
)

// routeTableOps takes the old and new routes from flatmap.Expand
// and returns a set of operations that must be performed in order
// to get to the desired state.
func routeTableOps(a interface{}, b interface{}) []routeTableOp {
	// Build up the actual ec2.Route objects
	oldRoutes := make(map[string]ec2.Route)
	newRoutes := make(map[string]ec2.Route)
	for i, raws := range []interface{}{a, b} {
		result := oldRoutes
		if i == 1 {
			result = newRoutes
		}
		if raws == nil {
			continue
		}

		for _, raw := range raws.([]interface{}) {
			m := raw.(map[string]interface{})
			r := ec2.Route{
				DestinationCidrBlock: m["cidr_block"].(string),
			}
			if v, ok := m["gateway_id"]; ok {
				r.GatewayId = v.(string)
			}
			if v, ok := m["instance_id"]; ok {
				r.InstanceId = v.(string)
			}

			result[r.DestinationCidrBlock] = r
		}
	}

	// Now, start building up the ops
	ops := make([]routeTableOp, 0, len(newRoutes))
	for n, r := range newRoutes {
		op := routeTableOpCreate
		if oldR, ok := oldRoutes[n]; ok {
			if reflect.DeepEqual(r, oldR) {
				// No changes!
				continue
			}

			op = routeTableOpReplace
		}

		ops = append(ops, routeTableOp{
			Op:    op,
			Route: r,
		})
	}

	// Determine what routes we need to delete
	for _, op := range ops {
		delete(oldRoutes, op.Route.DestinationCidrBlock)
	}
	for _, r := range oldRoutes {
		ops = append(ops, routeTableOp{
			Op:    routeTableOpDelete,
			Route: r,
		})
	}

	return ops
}

// RouteTableStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a RouteTable.
func RouteTableStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeRouteTables([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on RouteTableStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		rt := &resp.RouteTables[0]
		return rt, "ready", nil
	}
}
