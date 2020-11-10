package funcs

import (
	"fmt"
	"math/big"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"
)

// CidrHostFunc contructs a function that calculates a full host IP address
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
		var hostNum *big.Int
		if err := gocty.FromCtyValue(args[1], &hostNum); err != nil {
			return cty.UnknownVal(cty.String), err
		}
		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("invalid CIDR expression: %s", err)
		}

		ip, err := cidr.HostBig(network, hostNum)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(ip.String()), nil
	},
})

// CidrNetmaskFunc contructs a function that converts an IPv4 address prefix given
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

// CidrSubnetFunc contructs a function that calculates a subnet address within
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
		var netnum *big.Int
		if err := gocty.FromCtyValue(args[2], &netnum); err != nil {
			return cty.UnknownVal(cty.String), err
		}

		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), fmt.Errorf("invalid CIDR expression: %s", err)
		}

		newNetwork, err := cidr.SubnetBig(network, newbits, netnum)
		if err != nil {
			return cty.UnknownVal(cty.String), err
		}

		return cty.StringVal(newNetwork.String()), nil
	},
})

// CidrSubnetsFunc is similar to CidrSubnetFunc but calculates many consecutive
// subnet addresses at once, rather than just a single subnet extension.
var CidrSubnetsFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name: "prefix",
			Type: cty.String,
		},
	},
	VarParam: &function.Parameter{
		Name: "newbits",
		Type: cty.Number,
	},
	Type: function.StaticReturnType(cty.List(cty.String)),
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		_, network, err := net.ParseCIDR(args[0].AsString())
		if err != nil {
			return cty.UnknownVal(cty.String), function.NewArgErrorf(0, "invalid CIDR expression: %s", err)
		}
		startPrefixLen, _ := network.Mask.Size()

		prefixLengthArgs := args[1:]
		if len(prefixLengthArgs) == 0 {
			return cty.ListValEmpty(cty.String), nil
		}

		var firstLength int
		if err := gocty.FromCtyValue(prefixLengthArgs[0], &firstLength); err != nil {
			return cty.UnknownVal(cty.String), function.NewArgError(1, err)
		}
		firstLength += startPrefixLen

		retVals := make([]cty.Value, len(prefixLengthArgs))

		current, _ := cidr.PreviousSubnet(network, firstLength)
		for i, lengthArg := range prefixLengthArgs {
			var length int
			if err := gocty.FromCtyValue(lengthArg, &length); err != nil {
				return cty.UnknownVal(cty.String), function.NewArgError(i+1, err)
			}

			if length < 1 {
				return cty.UnknownVal(cty.String), function.NewArgErrorf(i+1, "must extend prefix by at least one bit")
			}
			// For portability with 32-bit systems where the subnet number
			// will be a 32-bit int, we only allow extension of 32 bits in
			// one call even if we're running on a 64-bit machine.
			// (Of course, this is significant only for IPv6.)
			if length > 32 {
				return cty.UnknownVal(cty.String), function.NewArgErrorf(i+1, "may not extend prefix by more than 32 bits")
			}
			length += startPrefixLen
			if length > (len(network.IP) * 8) {
				protocol := "IP"
				switch len(network.IP) * 8 {
				case 32:
					protocol = "IPv4"
				case 128:
					protocol = "IPv6"
				}
				return cty.UnknownVal(cty.String), function.NewArgErrorf(i+1, "would extend prefix to %d bits, which is too long for an %s address", length, protocol)
			}

			next, rollover := cidr.NextSubnet(current, length)
			if rollover || !network.Contains(next.IP) {
				// If we run out of suffix bits in the base CIDR prefix then
				// NextSubnet will start incrementing the prefix bits, which
				// we don't allow because it would then allocate addresses
				// outside of the caller's given prefix.
				return cty.UnknownVal(cty.String), function.NewArgErrorf(i+1, "not enough remaining address space for a subnet with a prefix of %d bits after %s", length, current.String())
			}

			current = next
			retVals[i] = cty.StringVal(current.String())
		}

		return cty.ListVal(retVals), nil
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

// CidrSubnets calculates a sequence of consecutive subnet prefixes that may
// be of different prefix lengths under a common base prefix.
func CidrSubnets(prefix cty.Value, newbits ...cty.Value) (cty.Value, error) {
	args := make([]cty.Value, len(newbits)+1)
	args[0] = prefix
	copy(args[1:], newbits)
	return CidrSubnetsFunc.Call(args)
}
