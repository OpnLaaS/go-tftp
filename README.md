# Go TFTP

A simple TFTP server implementation in Go for the OpnLaaS project.

`go get github.com/OpnLaaS/go-tftp`

```go
package main

import (
	"fmt"

	tftp "github.com/OpnLaaS/go-tftp"
)

func main() {
	fmt.Println("Starting server")

	ch, err := tftp.Serve()

	if err != nil {
		fmt.Println(err)
		return
	}

	<-ch
}
```
