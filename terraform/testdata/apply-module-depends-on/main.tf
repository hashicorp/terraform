module "moda" {
  source = "./moda"
  depends_on = [test_instance.a, module.modb]
}

resource "test_instance" "a" {
  depends_on = [module.modb]
  num = 4
  foo = test_instance.aa.id
}

resource "test_instance" "aa" {
  num = 3
  foo = module.modb.out
}

module "modb" {
  source = "./modb"
  depends_on = [test_instance.b]
}

resource "test_instance" "b" {
  num = 1
}

output "moda_data" {
  value = module.moda.out
}

output "modb_resource" {
  value = module.modb.out
}
