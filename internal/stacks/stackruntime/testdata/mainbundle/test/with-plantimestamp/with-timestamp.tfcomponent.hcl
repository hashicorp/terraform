component "self" {
  source = "./"
  inputs = {
      value = plantimestamp()
  }
}

component "second-self" {
  source = "./"
  inputs = {
      value = plantimestamp()
  }
}

output "plantimestamp" {
  type = string
  value = plantimestamp()
}
