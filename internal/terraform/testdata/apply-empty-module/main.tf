module "child" {
    source = "./child"
}

output "end" {
    value = "${module.child.aws_route53_zone_id}"
}
