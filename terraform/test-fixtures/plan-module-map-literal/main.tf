module "child" {
  source = "./child"
  amap {
    foo = "bar"
  }
  othermap {}
}
