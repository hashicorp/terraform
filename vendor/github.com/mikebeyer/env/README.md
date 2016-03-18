env
===

[ ![Codeship Status for mikebeyer/env](https://codeship.io/projects/d046ac90-ae6d-0132-79aa-6a5d0765ab36/status)](https://codeship.io/projects/68901)


Go implementation for default values for environment variables.

~~~ go
package main

import "github.com/mikebeyer/env"

func main() {
  port := env.String("PORT", "8080")
  fmt.Printf("port: %s", port)
}
~~~
