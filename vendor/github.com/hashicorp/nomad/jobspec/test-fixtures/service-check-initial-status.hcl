job "check_initial_status" {

    type = "service"
    group "group" {
        count = 1

        task "task" {
          service {
            tags = ["foo", "bar"]
            port = "http"

            check {
              name     = "check-name"
              type     = "http"
              interval = "10s"
              timeout  = "2s"
              initial_status = "passing"
            }
          }
        }
    }
}

