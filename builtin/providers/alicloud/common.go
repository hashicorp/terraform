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
const defaultLongTimeout = 1000

func getRegion(d *schema.ResourceData, meta interface{}) common.Region {
	return meta.(*AliyunClient).Region
}

func notFoundError(err error) bool {
	if e, ok := err.(*common.Error); ok &&
		(e.StatusCode == 404 || e.ErrorResponse.Message == "Not found" || e.Code == InstanceNotfound) {
		return true
	}

	return false
}

// Protocol represents network protocol
type Protocol string

// Constants of protocol definition
const (
	Http  = Protocol("http")
	Https = Protocol("https")
	Tcp   = Protocol("tcp")
	Udp   = Protocol("udp")
)

// ValidProtocols network protocol list
var ValidProtocols = []Protocol{Http, Https, Tcp, Udp}

// simple array value check method, support string type only
func isProtocolValid(value string) bool {
	res := false
	for _, v := range ValidProtocols {
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

const COLON_SEPARATED = ":"

const LOCAL_HOST_IP = "127.0.0.1"

// Takes the result of flatmap.Expand for an array of strings
// and returns a []string
func expandStringList(configured []interface{}) []string {
	vs := make([]string, 0, len(configured))
	for _, v := range configured {
		vs = append(vs, v.(string))
	}
	return vs
}
