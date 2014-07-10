package aws

import (
	"fmt"
	"log"
	"reflect"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/diff"
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

		// Append to the routes what we've done so far
		resultRoutes = append(resultRoutes, map[string]string{
			"cidr_block":  op.Route.DestinationCidrBlock,
			"gateway_id":  op.Route.GatewayId,
			"instance_id": op.Route.InstanceId,
		})
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

	log.Printf("[INFO] Deleting Route Table: %s", s.ID)
	if _, err := ec2conn.DeleteRouteTable(s.ID); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table: %s", err)
	}

	return nil
}

func resource_aws_route_table_refresh(
	s *terraform.ResourceState,
	meta interface{}) (*terraform.ResourceState, error) {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	resp, err := ec2conn.DescribeRouteTables([]string{s.ID}, ec2.NewFilter())
	if err != nil {
		if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil, nil
		}

		log.Printf("[ERROR] Error searching for route table: %s", err)
		return s, err
	}

	if len(resp.RouteTables) == 0 {
		return nil, nil
	}

	rt := &resp.RouteTables[0]
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
