package main

import (
	"log"
	"net/http"

	"github.com/cocaine/cocaine-framework-go/cocaine12/proxy"
)

func main() {
	err := http.ListenAndServe(":10000", proxy.NewServer())
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
