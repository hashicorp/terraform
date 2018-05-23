package funcs

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

//  CidrHostFunc contructs a function that calculates a full host IP address
// within a given IP network address prefix.
var CidrHostFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "prefix",
			Type: cty.String,
		},
		{
			Name: "hostnum",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var hostNum int
		if err := gocty.FromCtyValue(args[1], &hostNum); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("invalid CIDR expression: %s", err)
		}

		ip, err := cidr.Host(network, hostNum)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(ip.String()), nil
	},
})

//  CidrNetmaskFunc contructs a function that converts an IPv4 address prefix given
// in CIDR notation into a subnet mask address.
var CidrNetmaskFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "prefix",
			Type: cty.String,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("invalid CIDR expression: %s", err)
		}

		return cty.StringVal(net.IP(network.Mask).String()), nil
	},
})

//  CidrSubnetFunc contructs a function that calculates a subnet address within
// a given IP network address prefix.
var CidrSubnetFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "prefix",
			Type: cty.String,
		},
		{
			Name: "newbits",
			Type: cty.Number,
		},
		{
			Name: "netnum",
			Type: cty.Number,
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		var newbits int
		if err := gocty.FromCtyValue(args[1], &newbits); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		var netnum int
		if err := gocty.FromCtyValue(args[2], &netnum); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("invalid CIDR expression: %s", err)
		}

		// For portability with 32-bit systems where the subnet number
		// will be a 32-bit int, we only allow extension of 32 bits in
		// one call even if we're running on a 64-bit machine.
		// (Of course, this is significant only for IPv6.)
		if newbits > 32 {
			return cty.UnknownVal(cty.String), fmt.Errorf("may not extend prefix by more than 32 bits")
		}

		newNetwork, err := cidr.Subnet(network, newbits, netnum)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(newNetwork.String()), nil
	},
})

// CidrHost calculates a full host IP address within a given IP network address prefix.
func CidrHost(prefix, hostnum cty.Value) (cty.Value, error) {
	return CidrHostFunc.Call([]cty.Value{prefix, hostnum})
}

// CidrNetmask converts an IPv4 address prefix given in CIDR notation into a subnet mask address.
func CidrNetmask(prefix cty.Value) (cty.Value, error) {
	return CidrNetmaskFunc.Call([]cty.Value{prefix})
}

// CidrSubnet calculates a subnet address within a given IP network address prefix.
func CidrSubnet(prefix, newbits, netnum cty.Value) (cty.Value, error) {
	return CidrSubnetFunc.Call([]cty.Value{prefix, newbits, netnum})
}
