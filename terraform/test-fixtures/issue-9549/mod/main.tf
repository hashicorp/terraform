resource "template_file" "example" {
  template = "template text"
}

output "base_config" {
  value = {
    base_template = "${template_file.example.rendered}"

    # without this we fail with no entries
    extra = "value"
  }
}
