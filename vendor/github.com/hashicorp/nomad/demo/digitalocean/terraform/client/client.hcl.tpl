datacenter = "${datacenter}"
client {
    enabled = true
    servers = [${join(",", formatlist("\"%s:4647\"", servers))}]
    node_class = "linux-64bit"
}
