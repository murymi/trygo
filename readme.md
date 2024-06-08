#### server
```go
package main

import (
	"fmt"
	"io"
	"net"

	"github.com/wojciechmurimi/trygo"
)

func main() {
 	var l, err = net.Listen("tcp", ":3000")
 	if err != nil {
 		panic("failed to create tcp listener")
 	}

 	var server = trygo.CreateHTTPServer(l)

 	for {
 		request, err := server.Accept()
 		if err != nil {
 			panic("failed to accept a new connection")
 		}
 
 		fmt.Println("a new connection has been accepted")
 		var buff [1024]byte
 
 		for !request.Finished() {
 			_, e := request.Read(buff[:])
 
 			if e != nil {
 				if e != io.EOF {
 					panic(e)
 				}
                break
 			}
 		}
 		var response trygo.ResponseBuilder
		response.SetCode(200).SetHeader("done", "yes")
 		request.Write([]byte(response.String()))
 		request.Close()
 	}
}
```

#### client

```go
package main

import (
	"fmt"
	"io"

	"github.com/wojciechmurimi/trygo"
)

func main() {
	var client trygo.HTTPClient
	resp, err := client.Connect("https://internet.org")
	if err != nil {
		fmt.Println(err);
		panic("failed to connect")
	}
	var buff [1024]byte
	for !resp.Finished() {
		r, e := resp.Read(buff[:])
		if e != nil {
			if e != io.EOF {
				panic(e)
			}
			break
		}
		fmt.Println(string(buff[:r]))
	}
}
```