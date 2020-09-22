provider "template" {

}

data "template_file" "test" {
  template = "Hello World"
}
