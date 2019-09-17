variable "input" {
    type = "string"
    default = "hello world"
}

module "test" {
    source = "./inner"

    map_in = "${var.input}"
}   
