output "nginx_url" {
  value = "${element(split(",", alicloud_nat_gateway.default.bandwidth_packages.0.public_ip_addresses),1)}:80/test.php"
}