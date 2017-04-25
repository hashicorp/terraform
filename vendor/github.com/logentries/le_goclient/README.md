# Logentries client (golang)
Provides the capability to perform CRUD operations on log sets, logs, and log types.

# Example

```
package main

import (
   "fmt"
   logentries "github.com/logentries/le_goclient"
)

func main() {
   client := logentries.NewClient("<account_key>")
   res, err := client.User.Read(logentries.UserReadRequest{})
   fmt.Printf("err: %s\n", err)
   fmt.Println(res)
}
```

# License
See LICENSE.md
