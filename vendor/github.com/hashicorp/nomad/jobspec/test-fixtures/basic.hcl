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
      mode     = "delay"
    }

    task "binstore" {
      driver = "docker"
      user   = "bob"

      config {
        image = "hashicorp/binstore"

        labels {
          FOO = "bar"
        }
      }

      logs {
        max_files     = 10
        max_file_size = 100
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
          type     = "tcp"
          interval = "10s"
          timeout  = "2s"
          port     = "admin"
        }
      }

      resources {
        cpu    = 500
        memory = 128

        network {
          mbits = "100"

          port "one" {
            static = 1
          }

          port "two" {
            static = 2
          }

          port "three" {
            static = 3
          }

          port "http" {
          }

          port "https" {
          }

          port "admin" {
          }
        }
      }

      kill_timeout = "22s"

      artifact {
        source = "http://foo.com/artifact"

        options {
          checksum = "md5:b8a4f3f72ecab0510a6a31e997461c5f"
        }
      }

      artifact {
        source = "http://bar.com/artifact"

        options {
          checksum = "md5:ff1cc0d3432dad54d607c1505fb7245c"
        }
      }

      vault {
        policies = ["foo", "bar"]
      }
    }

    task "storagelocker" {
      driver = "docker"

      config {
        image = "hashicorp/storagelocker"
      }

      resources {
        cpu    = 500
        memory = 128
        iops   = 30
      }

      constraint {
        attribute = "kernel.arch"
        value     = "amd64"
      }
    }

    constraint {
      attribute = "kernel.os"
      value     = "linux"
    }

    meta {
      elb_mode     = "tcp"
      elb_interval = 10
      elb_checks   = 3
    }
  }
}
