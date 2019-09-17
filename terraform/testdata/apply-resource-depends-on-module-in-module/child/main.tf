module "grandchild" {
  source = "./child"
}

resource "aws_instance" "b" {
  ami        = "child"
  depends_on = ["module.grandchild"]
}
