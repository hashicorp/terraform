variable "foo" {
  default = <<EOT
Interpolation sequences $${are not allowed} in here.
EOT

}
