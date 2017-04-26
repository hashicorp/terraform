resource "alicloud_slb" "instance" {
  name = "${var.slb_name}"
  internet_charge_type = "${var.internet_charge_type}"
  internet = "${var.internet}"

  listener = [
    {
        "instance_port" = "22"
        "lb_port" = "22"
        "lb_protocol" = "tcp"
        "bandwidth" = "10"
        "health_check_type" = "http"
        "persistence_timeout" = 3600
        "healthy_threshold" = 8
        "unhealthy_threshold" = 8
        "health_check_timeout" = 8
        "health_check_interval" = 5
        "health_check_http_code" = "http_2xx,http_3xx"
        "health_check_timeout" = 8
    },

    {
      "instance_port" = "2001"
      "lb_port" = "2001"
      "lb_protocol" = "udp"
      "bandwidth" = "10"
      "persistence_timeout" = 3600
      "healthy_threshold" = 8
      "unhealthy_threshold" = 8
      "health_check_timeout" = 8
      "health_check_interval" = 4
      "health_check_timeout" = 8
    },

    {
        "instance_port" = "80"
        "lb_port" = "80"
        "lb_protocol" = "http"
        "sticky_session" = "on"
        "sticky_session_type" = "server"
        "cookie" = "testslblistenercookie"
        "cookie_timeout" = 86400
        "health_check" = "on"
        "health_check_domain" = "$_ip"
        "health_check_uri" = "/console"
        "health_check_connect_port" = 20
        "healthy_threshold" = 8
        "unhealthy_threshold" = 8
        "health_check_timeout" = 8
        "health_check_interval" = 5
        "health_check_http_code" = "http_2xx,http_3xx"
        "bandwidth" = 10
      }]
}
