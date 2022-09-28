resource "aws_instance" "a" {
    foo = "bar"
}

module "child" {
  source = "./child"
  in = aws_instance.a.id
}

output "out" {
  value = aws_instance.a.id
}
