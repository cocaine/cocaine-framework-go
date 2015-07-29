package trace

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/net/context"
)

const NullId = 0

type Id int64

type traced struct {
	context.Context
	parentId  Id
	spanId    Id
	traceId   Id
	startTime time.Time
}

func NewTraced() context.Context {
	return &traced{
		Context:   context.Background(),
		parentId:  0,
		spanId:    0,
		traceId:   0,
		startTime: time.Now(),
	}
}

func (t *traced) Value(key interface{}) interface{} {
	switch key {
	case "trace.parentid":
		return t.parentId
	case "trace.spanid":
		return t.spanId
	case "trace.traceid":
		return t.traceId
	case "traced.start":
		return t.startTime
	default:
		return t.Context.Value(key)
	}
}

func getId(ctx context.Context, name string) Id {
	if val, ok := ctx.Value(name).(Id); ok {
		return val
	}

	return NullId
}

func getParentId(ctx context.Context) Id {
	return getId(ctx, "trace.spanid")
}

func getTraceId(ctx context.Context) Id {
	return getId(ctx, "trace.traceid")
}

func WithTrace(ctx context.Context) (context.Context, func(format string, args ...interface{})) {
	if ctx == nil {
		ctx = context.Background()
	}

	parentId := getParentId(ctx)
	spanId := Id(rand.Int63())
	traceId := getTraceId(ctx)
	startTime := time.Now()

	ctx = &traced{
		Context:   ctx,
		parentId:  parentId,
		spanId:    spanId,
		traceId:   traceId,
		startTime: startTime,
	}

	return ctx, func(format string, args ...interface{}) {
		fmt.Printf("%v %v %v %v %s\n", startTime.Unix(),
			traceId, parentId, spanId, fmt.Sprintf(format, args...))
	}
}
