job "binstore-storagelocker" {
  datacenters = ["us2", "eu1"]

  group "binsl" {
    task "binstore" {
      driver = "docker"
      user   = "bob"

      config {
        image = "hashicorp/binstore"
      }

      logs {
        max_files     = 10
        max_file_size = 100
      }

      vault {
        policies = ["foo", "bar"]
      }
      vault {
        policies = ["1", "2"]
      }
    }
}
