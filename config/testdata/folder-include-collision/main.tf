folder {
  source = "./child-folder-1"
}

folder {
  source = "./child-folder-2"
}

output "a" {
  value = "${var.a}"
}

output "b" {
  value = "${var.b}"
}