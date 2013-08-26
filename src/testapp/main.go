package main

import (
	"cocaine"
)

func dummy(request *cocaine.Request, response *cocaine.Response) {
	logger := cocaine.NewLogger()
	logger.Info("FROM HANDLER")
	<-request.Read()
	logger.Info("AFTER READ")
	response.Write("Hello! I'm GO worker")
	logger.Info("AFTER WRITE")
	response.Close()
	logger.Info("AFTER CLOSE")
}

func echo(request *cocaine.Request, response *cocaine.Response) {
	inc := <-request.Read()
	response.Write(inc)
	response.Close()
}

func main() {
	binds := map[string]cocaine.EventHandler{
		"testevent": dummy,
		"echo":      echo}
	Worker := cocaine.NewWorker()
	Worker.Loop(binds)
}
