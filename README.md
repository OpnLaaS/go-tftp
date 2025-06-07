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

	quit, err := tftp.Serve(TFTPOptions{
		RootDir: "/var/tftp",
		TFTP_Address: ":69",

		ServeHTTP: true,
		HTTP_RootDir: "/var/www",
		HTTP_Address: ":8069"
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	<-quit
}
```
