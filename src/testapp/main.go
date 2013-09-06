package main

import (
	"cocaine"
	"fmt"
	"strings"
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

func app_list(request *cocaine.Request, response *cocaine.Response) {
	logger := cocaine.NewLogger()
	<-request.Read()
	storage, err := cocaine.NewStorage("localhost", 10053)
	if err != nil {
		logger.Err(fmt.Sprintf("Unable to create storage %s", err))
	}
	if list := <-storage.Find("apps", []string{"app"}); list.Err != nil {
		logger.Err(fmt.Sprintf("Unable to fetch  list%s", list.Err))
		response.Write("fail")
		response.Close()
	} else {
		response.Write(strings.Join(list.Res, ","))
		response.Close()
	}
}

func main() {
	binds := map[string]cocaine.EventHandler{
		"testevent": dummy,
		"echo":      echo,
		"test":      app_list}
	Worker := cocaine.NewWorker()
	Worker.Loop(binds)
}
