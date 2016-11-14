# Increase log verbosity
log_level = "DEBUG"

# Setup data dir
data_dir = "/tmp/client2"

# Enable the client
client {
    enabled = true

    # For demo assume we are talking to server1. For production,
    # this should be like "nomad.service.consul:4647" and a system
    # like Consul used for service discovery.
    servers = ["127.0.0.1:4647"]

    # Set ourselves as thing one
    meta {
        ssd = "true"
    }
}

# Modify our port to avoid a collision with server1 and client1
ports {
    http = 5657
}
