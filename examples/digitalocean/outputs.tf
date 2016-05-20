output "Public ip" {
	value = "${digitalocean_droplet.mywebserver.ipv4_address}"
}

output "Name" {
	value = "${digitalocean_droplet.mywebserver.name}"
}
