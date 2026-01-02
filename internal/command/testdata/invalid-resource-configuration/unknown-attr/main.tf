resource "test_instance" "my-resource" {
  ami   = "hello"
  amigo = "world" # unknown, not in the schema
}
