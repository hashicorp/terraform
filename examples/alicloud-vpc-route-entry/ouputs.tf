output "route_table_id" {
  value = "${alicloud_route_entry.default.route_table_id}"
}

output "router_id" {
  value = "${alicloud_route_entry.default.router_id}"
}

output "nexthop_type" {
  value = "${alicloud_route_entry.default.nexthop_type}"
}

output "nexthop_id" {
  value = "${alicloud_route_entry.default.nexthop_id}"
}