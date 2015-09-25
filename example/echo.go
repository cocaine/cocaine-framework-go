package main

import (
	"fmt"
	"github.com/cocaine/cocaine-framework-go/cocaine12"
	"golang.org/x/net/context"
	"time"
)

func Echo(ctx context.Context, req cocaine12.Request, resp cocaine12.Response) {
	defer resp.Close()

	ctx, done := cocaine12.WithTrace(ctx, "echo")
	defer done("Echo has finished")

	body, err := req.Read(ctx)
	if err != nil {
		resp.ErrorMsg(999, err.Error())
		return
	}

	time.Sleep(time.Millisecond * 100)
	resp.Write(body)
}

func main() {
	w, err := cocaine12.NewWorker()
	if err != nil {
		panic(err)
	}

	w.SetTerminationHandler(func(ctx context.Context) {
		time.Sleep(60 * time.Millisecond)
	})

	w.EnableStackSignal(false)
	w.On("ping", Echo)

	if err = w.Run(nil); err != nil {
		fmt.Printf("%v", err)
	}
}
