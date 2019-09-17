locals {
  baz          = { "greeting" = "hello" }
  cert_options = <<EOF
    A
    B ${lookup(local.baz, "greeting")}
    C
  EOF
}

output "local" {
  value = "${local.cert_options}"
}
