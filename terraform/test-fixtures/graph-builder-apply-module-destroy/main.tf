variable "input" { default = "value" }

module "A" {
    source = "./A"
    input  = "${var.input}"
}

module "B" {
    source = "./A"
    input  = "${module.A.output}"
}
