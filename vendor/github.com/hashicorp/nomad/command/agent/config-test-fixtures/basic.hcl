region = "foobar"
datacenter = "dc2"
name = "my-web"
data_dir = "/tmp/nomad"
log_level = "ERR"
bind_addr = "192.168.0.1"
enable_debug = true
ports {
	http = 1234
	rpc = 2345
	serf = 3456
}
addresses {
	http = "127.0.0.1"
	rpc = "127.0.0.2"
	serf = "127.0.0.3"
}
advertise {
	rpc = "127.0.0.3"
	serf = "127.0.0.4"
}
client {
	enabled = true
	state_dir = "/tmp/client-state"
	alloc_dir = "/tmp/alloc"
	servers = ["a.b.c:80", "127.0.0.1:1234"]
	node_class = "linux-medium-64bit"
	meta {
		foo = "bar"
		baz = "zip"
	}
	options {
		foo = "bar"
		baz = "zip"
	}
	chroot_env {
		"/opt/myapp/etc" = "/etc"
		"/opt/myapp/bin" = "/bin"
	}
	network_interface = "eth0"
	network_speed = 100
	reserved {
		cpu = 10
		memory = 10
		disk = 10
		iops = 10
		reserved_ports = "1,100,10-12"
	}
	client_min_port = 1000
	client_max_port = 2000
    max_kill_timeout = "10s"
    stats {
        data_points = 35
        collection_interval = "5s"
    }
}
server {
	enabled = true
	bootstrap_expect = 5
	data_dir = "/tmp/data"
	protocol_version = 3
	num_schedulers = 2
	enabled_schedulers = ["test"]
	node_gc_threshold = "12h"
	heartbeat_grace   = "30s"
	retry_join = [ "1.1.1.1", "2.2.2.2" ]
	start_join = [ "1.1.1.1", "2.2.2.2" ]
	retry_max = 3
	retry_interval = "15s"
	rejoin_after_leave = true
}
telemetry {
	statsite_address = "127.0.0.1:1234"
	statsd_address = "127.0.0.1:2345"
	disable_hostname = true
    collection_interval = "3s"
    publish_allocation_metrics = true
    publish_node_metrics = true
}
leave_on_interrupt = true
leave_on_terminate = true
enable_syslog = true
syslog_facility = "LOCAL1"
disable_update_check = true
disable_anonymous_signature = true
atlas {
	infrastructure = "armon/test"
	token = "abcd"
	join = true
	endpoint = "127.0.0.1:1234"
}
http_api_response_headers {
	Access-Control-Allow-Origin = "*"
}
consul {
    server_service_name = "nomad"
    client_service_name = "nomad-client"
    address = "127.0.0.1:9500"
    token = "token1"
    auth = "username:pass"
    ssl = true
    verify_ssl = false
    ca_file = "/path/to/ca/file"
    cert_file = "/path/to/cert/file"
    key_file = "/path/to/key/file"
    server_auto_join = false
    client_auto_join = false
    auto_advertise = false
}
