job "binstore-storagelocker" {
  region      = "global"
  type        = "service"
  priority    = 50
  all_at_once = true
  datacenters = ["us2", "eu1"]
  vault_token = "foo"

  meta {
    foo = "bar"
  }

  constraint {
    attribute = "kernel.os"
    value     = "windows"
  }

  update {
    stagger      = "60s"
    max_parallel = 2
  }

  task "outside" {
    driver = "java"

    config {
      jar_path = "s3://my-cool-store/foo.jar"
    }

    meta {
      my-cool-key = "foobar"
    }
  }

  group "binsl" {
    count = 5

    restart {
      attempts = 5
      interval = "10m"
      delay    = "15s"
    }

    task "binstore" {
      driver = "docker"

      config {
        image = "hashicorp/binstore"
      }

      env {
        HELLO = "world"
        LOREM = "ipsum"
      }

      service {
        tags = ["foo", "bar"]
        port = "http"

        check {
          name     = "check-name"
          type     = "http"
          interval = "10s"
          timeout  = "2s"
        }
      }

      service {
        port = "one"
      }

      resources {
        cpu    = 500
        memory = 128

        network {
          mbits = "100"

          port "one" {
            static = 1
          }

          port "three" {
            static = 3
          }

          port "http" {
          }
        }
      }
    }
  }
}
