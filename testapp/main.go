package main

import (
	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"golang.org/x/net/context"
)

func hanlder(ctx context.Context, req cocaine.Request, resp cocaine.Response) {
	defer resp.Close()

	data, err := req.Read()
	if err != nil {
		panic(err)
	}
	resp.Write(data)
}

func main() {
	w, err := cocaine.NewWorker()
	if err != nil {
		panic(err)
	}

	hanlders := map[string]cocaine.EventHandler{
		"echo": hanlder,
	}
	err = w.Run(hanlders)
	if err != nil {
		panic(err)
	}
}
