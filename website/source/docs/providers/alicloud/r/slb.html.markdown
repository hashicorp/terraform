---
layout: "alicloud"
page_title: "Alicloud: alicloud_slb"
sidebar_current: "docs-alicloud-resource-slb"
description: |-
  Provides an Application Load Banlancer resource.
---

# alicloud\_slb

Provides an Application Load Balancer resource.

## Example Usage

```
# Create a new load balancer for classic
resource "alicloud_slb" "classic" {
  name                 = "test-slb-tf"
  internet             = true
  internet_charge_type = "paybybandwidth"
  bandwidth            = 5

  listener = [
    {
      "instance_port" = "2111"
      "lb_port"       = "21"
      "lb_protocol"   = "tcp"
      "bandwidth"     = "5"
    },
    {
      "instance_port" = "8000"
      "lb_port"       = "80"
      "lb_protocol"   = "http"
      "bandwidth"     = "5"
    },
    {
      "instance_port" = "1611"
      "lb_port"       = "161"
      "lb_protocol"   = "udp"
      "bandwidth"     = "5"
    },
  ]
}

# Create a new load balancer for VPC
resource "alicloud_vpc" "default" {
  # Other parameters...
}

resource "alicloud_vswitch" "default" {
  # Other parameters...
}

resource "alicloud_slb" "vpc" {
  name       = "test-slb-tf"
  vswitch_id = "${alicloud_vswitch.default.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The name of the SLB. This name must be unique within your AliCloud account, can have a maximum of 80 characters,
must contain only alphanumeric characters or hyphens, such as "-","/",".","_", and must not begin or end with a hyphen. If not specified,
Terraform will autogenerate a name beginning with `tf-lb`.
* `internet` - (Optional, Forces New Resource) If true, the SLB addressType will be internet, false will be intranet, Default is false. If load balancer launched in VPC, this value must be "false".
* `internet_charge_type` - (Optional, Forces New Resource) Valid
  values are `paybybandwidth`, `paybytraffic`. If this value is "paybybandwidth", then argument "internet" must be "true". Default is "paybytraffic". If load balancer launched in VPC, this value must be "paybytraffic".
* `bandwidth` - (Optional) Valid
  value is between 1 and 1000, If argument "internet_charge_type" is "paybytraffic", then this value will be ignore.
* `listener` - (Optional) Additional SLB listener. See [Block listener](#block-listener) below for details.
* `vswitch_id` - (Required for a VPC SLB, Forces New Resource) The VSwitch ID to launch in.

## Block listener

load balance support 4 protocal to listen on, they are `http`,`https`,`tcp`,`udp`, the every listener support which portocal following:

listener parameter | support protocol | value range |
------------- | ------------- | ------------- | 
instance_port | http & https & tcp & udp | 1-65535 | 
lb_port | http & https & tcp & udp | 1-65535 |
lb_protocol | http & https & tcp & udp |
bandwidth | http & https & tcp & udp | -1 / 1-1000 |
scheduler | http & https & tcp & udp | wrr or wlc |
sticky_session | http & https | on or off |
sticky_session_type | http & https | insert or server | 
cookie_timeout | http & https | 1-86400  | 
cookie | http & https |   | 
persistence_timeout | tcp & udp | 0-3600 | 
health_check | http & https | on or off | 
health_check_type | tcp | tcp or http | 
health_check_domain | http & https & tcp | 
health_check_uri | http & https & tcp |  | 
health_check_connect_port | http & https & tcp & udp | 1-65535 or -520 | 
healthy_threshold | http & https & tcp & udp | 1-10 | 
unhealthy_threshold | http & https & tcp & udp | 1-10 | 
health_check_timeout | http & https & tcp & udp | 1-50 |
health_check_interval | http & https & tcp & udp | 1-5 |
health_check_http_code | http & https & tcp | http_2xx,http_3xx,http_4xx,http_5xx | 
ssl_certificate_id | https |  |  


The listener mapping supports the following:

* `instance_port` - (Required) The port on which the backend servers are listening. Valid value is between 1 to 65535.
* `lb_port` - (Required) The port on which the load balancer is listening. Valid value is between 1 to 65535.
* `lb_protocol` - (Required) The protocol to listen on. Valid values are `http` and and `tcp` and `udp`.
* `bandwidth` - (Required) The bandwidth on which the load balancer is  listening. Valid values is -1 or between 1 and 1000. If -1, the bindwidth will haven’t upper limit.
* `scheduler` - (Optinal) Scheduling algorithm, Valid Value is `wrr` / `wlc`, Default is "wrr".
* `sticky_session` - (Optinal) Whether to enable session persistence, Value: `on` / `off`.
* `sticky_session_type` - (Optinal) Mode for handling the cookie. If "sticky_session" is on, the parameter is mandatory, and if "sticky_session" is off, the parameter will be ignored. Value：`insert` / `server`. If it is set to insert, it means it is inserted from Server Load Balancer; and if it is set to server, it means the Server Load Balancer learns from the backend server.
* `cookie_timeout` - (Optinal) The parameter is mandatory when "sticky_session" is on and "sticky_session_type" is insert. Otherwise, it will be ignored. Value： 1-86400（in seconds）
* `cookie` - (Optinal) The cookie configured on the server 
It is mandatory only when "sticky_session" is on and "sticky_session_type" is server; otherwise, the parameter will be ignored. Value：String in line with RFC 2965, with length being 1- 200. It only contains characters such as ASCII codes, English letters and digits instead of the comma, semicolon or spacing, and it cannot start with $.
* `persistence_timeout` - (Optinal) Timeout of connection persistence. Value： 0-3600（in seconds） .Default：0 The value 0 indicates to close it.
* `health_check` - (Optinal) Whether to enable health check. Value：`on` / `off`
* `health_check_type` - (Optinal) Type of health check. Value：`tcp` | `http` , Default：`tcp` . TCP supports TCP and HTTP health check mode, you can select the particular mode depending on your application.
* `health_check_domain` - (Optinal) Domain name used for health check. When TCP listener need to use HTTP health check, this parameter will be configured; and when TCP health check is used, the parameter will be ignored. Value： `$_ip | custom string`. Rules of the custom string: its length is limited to 1-80 and only characters such as letters, digits, ‘-‘ and ‘.’ are allowed. When the parameter is set to $_ip by the user, Server Load Balancer uses the private network IP address of each backend server as Domain used for health check.
* `health_check_uri` - (Optinal) URI used for health check. When TCP listener need to use HTTP health check, this parameter will be configured; and when TCP health check is used, the parameter will be ignored. 
Value：Its length is limited to 1-80 and it must start with /. Only characters such as letters, digits, ‘-’, ‘/’, ‘.’, ‘%’, ‘?’, #’ and ‘&’ are allowed.
* `health_check_connect_port` - (Optinal) Port used for health check. Value： `1-65535`, Default：None. When the parameter is not set, it means the backend server port is used (BackendServerPort).
* `healthy_threshold` - (Optinal) Threshold determining the result of the health check is success. Value：`1-10`, Default：3.
* `unhealthy_threshold` - (Optinal) Threshold determining the result of the health check is fail. Value：`1-10`, Default：3.
* `health_check_timeout` - (Optinal) Maximum timeout of each health check response. When "health_check" is on, the parameter is mandatory; and when "mandatory" is off, the parameter will be ignored. Value：`1-50`（in seconds）. Note: If health_check_timeout < health_check_interval, health_check_timeout is invalid, and the timeout is health_check_interval.
* `health_check_interval` - (Optinal) Time interval of health checks. 
When "health_check" is on, the parameter is mandatory; and when "health_check" is off, the parameter will be ignored. Value：`1-5` (in seconds）
* `health_check_http_code` - (Optinal) Regular health check HTTP status code. Multiple codes are segmented by “,”. When "health_check" is on, the parameter is mandatory; and when "health_check" is off, the parameter will be ignored.  Value：`http_2xx` / `http_3xx` / `http_4xx` / `http_5xx`.
* `ssl_certificate_id` - (Optinal) Security certificate ID.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the load balancer.
* `name` - The name of the load balancer.
* `internet` - The internet of the load balancer.
* `internet_charge_type` - The internet_charge_type of the load balancer.
* `bandwidth` - The bandwidth of the load balancer.
* `vswitch_id` - The VSwitch ID of the load balancer. Only available on SLB launched in a VPC.
* `address` - The IP address of the load balancer.