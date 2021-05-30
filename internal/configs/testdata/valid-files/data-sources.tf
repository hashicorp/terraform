data "http" "example1" {
}

data "http" "example2" {
  url = "http://example.com/"

  request_headers = {
    "Accept" = "application/json"
  }

  count = 5
  depends_on = [
    data.http.example1,
  ]
}
