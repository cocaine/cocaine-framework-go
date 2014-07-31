# cocaine-framework-Go
This package helps you to write golang application, which can be launched in PaaS Cocaine. 
You can write application for Cocaine so fast and easy as you cannot even imagine.

## A motivating example

This's a classical example of echo application:
```go
package main

import (
	"log"

	"github.com/cocaine/cocaine-framework-go/cocaine"
)


func echo(request *cocaine.Request, response *cocaine.Response) {
	inc := <-request.Read()
	response.Write(inc)
	response.Close()
}

func main() {
	binds := map[string]cocaine.EventHandler{
		"echo":      echo,
	}
	Worker, err := cocaine.NewWorker()
	if err != nil {
		log.Fatal(err)
	}
	Worker.Loop(binds)
}

```

This is an example of using go web frameworks:

```
package main

import (
	"github.com/cocaine/cocaine-framework-go/cocaine"
	"github.com/codegangsta/martini"
)


func main() {
	m := martini.Classic()
	m.Get("", func() string {
			return "This is an example server"
		})

	m.Get("/hw", func() string {
			return "Hello world!"
		})

	binds := map[string]cocaine.EventHandler{
		"example": cocaine.WrapHandler(m, nil),
	}
	if worker, err := cocaine.NewWorker(); err == nil{
		worker.Loop(binds)
	}else{
		panic(err)
	}
}
```

## Installation
```
go get github.com/cocaine/cocaine-framework-go/cocaine
```

## Package documentation

[Look at GoDoc](http://godoc.org/github.com/cocaine/cocaine-framework-go/cocaine)
