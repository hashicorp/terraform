package alicloud

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
)

type InstanceNetWork string

const (
	ClassicNet = InstanceNetWork("classic")
	VpcNet     = InstanceNetWork("vpc")
)

// timeout for common product, ecs e.g.
const defaultTimeout = 120

// timeout for long time progerss product, rds e.g.
const defaultLongTimeout = 800

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

var DefaultBusinessInfo = ecs.BusinessInfo{
	Pack: "terraform",
}

// default region for all resource
const DEFAULT_REGION = "cn-beijing"

// default security ip for db
const DEFAULT_DB_SECURITY_IP = "127.0.0.1"

// we the count of create instance is only one
const DEFAULT_INSTANCE_COUNT = 1

// symbol of multiIZ
const MULTI_IZ_SYMBOL = "MAZ"

// default connect port of db
const DB_DEFAULT_CONNECT_PORT = "3306"

const COMMA_SEPARATED = ","

const LOCAL_HOST_IP = "127.0.0.1"
