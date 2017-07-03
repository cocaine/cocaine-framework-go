package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cocaine/cocaine-framework-go/cocaine12"
)

func Echo(ctx context.Context, req cocaine12.Request, resp cocaine12.Response) {
	defer resp.Close()

	// Headers arrived with Invoke message
	_ = req.Headers()

	ctx, done := cocaine12.NewSpan(ctx, "echo")
	defer done()

	// Read call resets associated headers
	body, err := req.Read(ctx)
	if err != nil {
		resp.ErrorMsg(999, err.Error())
		return
	}
	// Headers arrived with next chunk
	h := req.Headers()

	time.Sleep(time.Millisecond * 100)
	// those headers will be sent with next Write
	resp.SetHeaders(h)
	resp.Write(body)

	h["A"] = []string{"B", "C"}
	// defer Close will sent those headers
	resp.SetHeaders(h)
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
