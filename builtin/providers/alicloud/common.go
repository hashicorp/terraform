package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/hashicorp/terraform/helper/schema"
)

type InstanceNetWork string

const (
	ClassicNet = InstanceNetWork("classic")
	VpcNet     = InstanceNetWork("vpc")
)

const defaultTimeout = 120

func getRegion(d *schema.ResourceData, meta interface{}) common.Region {
	return meta.(*AliyunClient).Region
}

func notFoundError(err error) bool {
	if e, ok := err.(*common.Error); ok && (e.StatusCode == 404 || e.ErrorResponse.Message == "Not found") {
		return true
	}

	return false
}

// Protocal represents network protocal
type Protocal string

// Constants of protocal definition
const (
	Http  = Protocal("http")
	Https = Protocal("https")
	Tcp   = Protocal("tcp")
	Udp   = Protocal("udp")
)

// ValidProtocals network protocal list
var ValidProtocals = []Protocal{Http, Https, Tcp, Udp}

// simple array value check method, support string type only
func isProtocalValid(value string) bool {
	res := false
	for _, v := range ValidProtocals {
		if string(v) == value {
			res = true
		}
	}
	return res
}
