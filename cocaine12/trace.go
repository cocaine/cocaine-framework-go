package cocaine12

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"golang.org/x/net/context"
)

var (
	initTraceLogger sync.Once
	traceLogger     Logger
)

func traceLog() Logger {
	initTraceLogger.Do(func() {
		var err error
		traceLogger, err = NewLogger(context.Background())
		// there must be no error
		if err != nil {
			panic(fmt.Sprintf("unable to create trace logger: %v", err))
		}
	})
	return traceLogger
}

type TraceInfo struct {
	trace, span, parent uint64
}

type traced struct {
	context.Context
	traceInfo TraceInfo
	startTime time.Time
}

func BeginNewTraceContext() context.Context {
	ts := uint64(rand.Int63())
	return NewTracedContext(TraceInfo{
		trace:  ts,
		span:   ts,
		parent: 0,
	})
}

func NewTracedContext(traceInfo TraceInfo) context.Context {
	return &traced{
		Context:   context.Background(),
		traceInfo: traceInfo,
		startTime: time.Now(),
	}
}

func (t *traced) Value(key interface{}) interface{} {
	switch key {
	case "trace.traceinfo":
		return t.traceInfo
	case "traced.start":
		return t.startTime
	default:
		return t.Context.Value(key)
	}
}

func getTraceInfo(ctx context.Context) *TraceInfo {
	if val, ok := ctx.Value("trace.traceinfo").(TraceInfo); ok {
		return &val
	}
	return nil
}

func nullDone(format string, args ...interface{}) {}

func WithTrace(ctx context.Context, rpcName string) (context.Context, func(format string, args ...interface{})) {
	if ctx == nil {
		return context.Background(), nullDone
	}

	traceInfo := getTraceInfo(ctx)
	if traceInfo == nil {
		return ctx, nullDone
	}

	startTime := time.Now()

	traceInfo.parent = traceInfo.span
	traceInfo.span = uint64(rand.Int63())
	traceLog().WithFields(Fields{
		"trace_id":  fmt.Sprintf("%x", traceInfo.trace),
		"span_id":   fmt.Sprintf("%x", traceInfo.span),
		"parent_id": fmt.Sprintf("%x", traceInfo.parent),
		"timestamp": time.Now().UnixNano(),
		"RPC":       rpcName,
	}).Infof("start")

	ctx = &traced{
		Context:   ctx,
		traceInfo: *traceInfo,
		startTime: startTime,
	}

	return ctx, func(format string, args ...interface{}) {
		traceLog().WithFields(Fields{
			"trace_id":  fmt.Sprintf("%x", traceInfo.trace),
			"span_id":   fmt.Sprintf("%x", traceInfo.span),
			"parent_id": fmt.Sprintf("%x", traceInfo.parent),
			"timestamp": time.Now().UnixNano(),
			"RPC":       rpcName,
		}).Infof(format, args...)
	}
}
