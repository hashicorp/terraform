variable "leader" {
    default = false
}

output "leader" {
    value = "${var.leader}"
}
