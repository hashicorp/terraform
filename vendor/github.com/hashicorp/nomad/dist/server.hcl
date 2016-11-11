bind_addr = "0.0.0.0"
data_dir = "/var/lib/nomad"

advertise {
	# This should be the IP of THIS MACHINE and must be routable by every node
	# in your cluster
	rpc = "1.2.3.4:4647"
}

server {
	enabled = true
	bootstrap_expect = 3
}