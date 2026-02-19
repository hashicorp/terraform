resource "null_resource" "noop" {}

module "test" {
    source = "./inner"

    im_a_string = "hello"
}
