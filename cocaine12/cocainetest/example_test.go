package cocainetest

import (
	"context"
	"fmt"

	cocaine "github.com/cocaine/cocaine-framework-go/cocaine12"
)

func Example() {
	var handler cocaine.EventHandler // check type

	handler = func(ctx context.Context, req cocaine.Request, resp cocaine.Response) {
		inp, _ := req.Read(ctx)

		resp.Write(inp)
		resp.ErrorMsg(100, "testerrormessage")
		resp.Close()
	}

	mockRequest := NewRequest()
	mockRequest.Write([]byte("PING"))

	mockResponse := NewResponse()

	handler(context.Background(), mockRequest, mockResponse)

	fmt.Printf("data: %s error: %d %s \n",
		mockResponse.Bytes(), mockResponse.Err.Code, mockResponse.Err.Msg)

	// Output: data: PING error: 100 testerrormessage
}
