bind_addr = "127.0.0.1"
data_dir = "/var/lib/nomad/"

client {
	enabled = true
	servers = ["10.1.0.1", "10.1.0.2", "10.1.0.3"]
}