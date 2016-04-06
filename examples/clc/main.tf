# --------------------
# Credentials
provider "clc" {
  username = "${var.clc_username}"
  password = "${var.clc_password}"
  account = "${var.clc_account}"
}

# --------------------
# Provision/Resolve a server group
resource "clc_group" "frontends" {
  location_id = "CA1"
  name = "frontends"
  parent = "Default Group"
}

# --------------------
# Provision a server
resource "clc_server" "node" {
  name_template = "trusty"
  source_server_id = "UBUNTU-14-64-TEMPLATE"
  group_id = "${clc_group.frontends.id}"
  cpu = 2
  memory_mb = 2048
  password = "Green123$"
  additional_disks
    {
        path = "/var"
        size_gb = 100
        type = "partitioned"
    }
  additional_disks
    {
        size_gb = 10
        type = "raw"
    }
}

# --------------------
# Provision a public ip
resource "clc_public_ip" "backdoor" {
  server_id = "${clc_server.node.0.id}"
  internal_ip_address = "${clc_server.node.0.private_ip_address}"
  ports
    {
      protocol = "ICMP"
      port = -1
    }
  ports
    {
      protocol = "TCP"
      port = 22
    }
  source_restrictions
     { cidr = "173.60.0.0/16" }


  # ssh in and start a simple http server on :8080
  provisioner "remote-exec" {
    inline = [
      "cd /tmp; python -mSimpleHTTPServer > /dev/null 2>&1 &"
    ]
    connection {
      host = "${clc_public_ip.backdoor.id}"
      user = "root"
      password = "${clc_server.node.password}"
    }
  }
  
}


# --------------------
# Provision a load balancer
resource "clc_load_balancer" "frontdoor" {
  data_center = "${clc_group.frontends.location_id}"
  name = "frontdoor"
  description = "frontdoor"
  status = "enabled"
}

# --------------------
# Provision a load balancer pool
resource "clc_load_balancer_pool" "pool" {
  data_center = "${clc_group.frontends.location_id}"
  load_balancer = "${clc_load_balancer.frontdoor.id}"
  method = "roundRobin"
  persistence = "standard"
  port = 80
  nodes
    {
      status = "enabled"
      ipAddress = "${clc_server.node.private_ip_address}"
      privatePort = 8000
    }
}
