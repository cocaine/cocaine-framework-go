package cocainetest

import (
	"fmt"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
	"golang.org/x/net/context"
)

func Example() {
	var handler cocaine.EventHandler // check type

	handler = func(ctx context.Context, req cocaine.Request, resp cocaine.Response) {
		inp, _ := req.Read()

		resp.Write(inp)
		resp.ErrorMsg(100, "testerrormessage")
		resp.Close()
	}

	mockRequest := NewRequest()
	mockRequest.Push([]byte("PING"))

	mockResponse := NewResponse()

	handler(context.Background(), mockRequest, mockResponse)

	fmt.Printf("data: %s error: %d %s \n",
		[]byte("PING"), mockResponse.Err.Code, mockResponse.Err.Msg)

	// Output: data: PING error: 100 testerrormessage
}
