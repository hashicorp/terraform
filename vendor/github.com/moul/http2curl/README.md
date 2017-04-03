# http2curl
:triangular_ruler: Convert Golang's http.Request to CURL command line

[![Build Status](https://travis-ci.org/moul/http2curl.svg?branch=master)](https://travis-ci.org/moul/http2curl)
[![GoDoc](https://godoc.org/github.com/moul/http2curl?status.svg)](https://godoc.org/github.com/moul/http2curl)
[![Coverage Status](https://coveralls.io/repos/moul/http2curl/badge.svg)](https://coveralls.io/github/moul/http2curl)

To do the reverse, check out [mholt/curl-to-go](https://github.com/mholt/curl-to-go).

## Example

```go
import "http"
import "github.com/moul/http2curl"

data := bytes.NewBufferString(`{"hello":"world","answer":42}`)
req, _ := http.NewRequest("PUT", "http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu", data)
req.Header.Set("Content-Type", "application/json")

command, _ := http2curl.GetCurlCommand(req)
fmt.Println(command)
// Output: curl -X PUT -d "{\"hello\":\"world\",\"answer\":42}" -H "Content-Type: application/json" http://www.example.com/abc/def.ghi?jlk=mno&pqr=stu
```

## Install

```php
$ go get github.com/moul/http2curl
```

## Usages

- https://github.com/parnurzeal/gorequest
- https://github.com/scaleway/scaleway-cli
- https://github.com/nmonterroso/cowsay-slackapp
- https://github.com/moul/as-a-service
- https://github.com/gavv/httpexpect
- https://github.com/smallnest/goreq

## License

MIT
