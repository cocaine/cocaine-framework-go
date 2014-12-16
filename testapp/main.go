package main

import (
	"log"

	"github.com/cocaine/cocaine-framework-go/cocaine"
)

var (
	logger *cocaine.Logger
	//storage *cocaine.Storage
)

func dummy(request cocaine.Request, response cocaine.Response) {
	defer response.Close()
	logger, err := cocaine.NewLogger()
	if err != nil {
		response.ErrorMsg(-1000, err.Error())
	}
	logger.Info("FROM HANDLER")
	<-request.Read()
	logger.Info("AFTER READ")
	response.Write("Hello! I'm GO worker")
	logger.Info("AFTER WRITE")
}

func echo(request cocaine.Request, response cocaine.Response) {
	inc := <-request.Read()
	response.Write(inc)
	response.Close()
}

func main() {
	var err error
	logger, err = cocaine.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	binds := map[string]cocaine.EventHandler{
		"testevent": dummy,
		"echo":      echo,
	}
	Worker, err := cocaine.NewWorker()
	if err != nil {
		log.Fatal(err)
	}
	Worker.Loop(binds)
}
