package main

import (
	"fmt"
	"github.com/cocaine/cocaine-framework-go/cocaine"
)

var (
	logger *cocaine.Logger
	//storage *cocaine.Storage
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
	<-request.Read()
	// if list := <-storage.Find("apps", []string{"app"}); list.Err != nil {
	// 	logger.Err(fmt.Sprintf("Unable to fetch  list%s", list.Err))
	// 	response.Write("fail")
	// 	response.Close()
	// } else {
	// 	response.Write(strings.Join(list.Res, ","))
	// 	response.Close()
	// }
}

func http_test(request *cocaine.Request, response *cocaine.Response) {
	req, err := cocaine.UnpackProxyRequest(<-request.Read())
	if err != nil {
		response.Write(cocaine.WriteHead(200, [][2]string{{"Content-Type", "text/html"}}))
		ans := fmt.Sprintf("Method: %s, Uri: %s, UA: %s", req.Method, req.URL, req.UserAgent())
		response.Write(ans)
	} else {
		logger.Err("Could not unpack request to http request")
	}
	response.Close()
}

func main() {
	logger = cocaine.NewLogger()
	//storage, _ = cocaine.NewStorage("localhost", 10053)
	binds := map[string]cocaine.EventHandler{
		"testevent": dummy,
		"echo":      echo,
		"test":      app_list,
		"http":      http_test}
	Worker := cocaine.NewWorker()
	Worker.Loop(binds)
}
