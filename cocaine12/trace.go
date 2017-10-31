package cocaine12

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	TraceInfoValue      = "trace.traceinfo"
	TraceStartTimeValue = "trace.starttime"
)

var (
	initTraceLogger sync.Once
	traceLogger     Logger

	closeDummySpan CloseSpan = func() {}
)

func GetTraceInfo(ctx context.Context) *TraceInfo {
	if val, ok := ctx.Value(TraceInfoValue).(TraceInfo); ok {
		return &val
	}
	return nil
}

// CloseSpan closes attached span. It should be call after
// the rpc ends.
type CloseSpan func()

type TraceInfo struct {
	Trace, Span, Parent uint64
	logger              Logger
}

func (traceInfo *TraceInfo) getLog() Logger {
	if traceInfo.logger != nil {
		return traceInfo.logger
	}

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

type traced struct {
	context.Context
	traceInfo TraceInfo
	startTime time.Time
}

func (t *traced) Value(key interface{}) interface{} {
	switch key {
	case TraceInfoValue:
		return t.traceInfo
	case TraceStartTimeValue:
		return t.startTime
	default:
		return t.Context.Value(key)
	}
}

// It might be used in client applications.
func BeginNewTraceContext(ctx context.Context) context.Context {
	return BeginNewTraceContextWithLogger(ctx, nil)
}

func BeginNewTraceContextWithLogger(ctx context.Context, logger Logger) context.Context {
	ts := uint64(rand.Int63())
	return AttachTraceInfo(ctx, TraceInfo{
		Trace:  ts,
		Span:   ts,
		Parent: 0,
		logger: logger,
	})
}

// AttachTraceInfo binds given TraceInfo to the context.
// If ctx is nil, then TraceInfo will be attached to context.Background()
func AttachTraceInfo(ctx context.Context, traceInfo TraceInfo) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return &traced{
		Context:   ctx,
		traceInfo: traceInfo,
		startTime: time.Now(),
	}
}

// CleanTraceInfo might be used to clear context instance from trace info
// to disable tracing in some RPC calls to get rid of overhead
func CleanTraceInfo(ctx context.Context) context.Context {
	return context.WithValue(ctx, TraceInfoValue, nil)
}

// NewSpan starts new span and returns a context with attached TraceInfo and Done.
// If ctx is nil or has no TraceInfo new span won't start to support sampling,
// so it's user responsibility to make sure that the context has TraceInfo.
// Anyway it safe to call CloseSpan function even in this case, it actually does nothing.
func NewSpan(ctx context.Context, rpcNameFormat string, args ...interface{}) (context.Context, CloseSpan) {
	if ctx == nil {
		// I'm not sure it is a valid action.
		// According to the rule "no trace info, no new span"
		// to support sampling, nil Context has no TraceInfo, so
		// it cannot start new Span.
		return context.Background(), closeDummySpan
	}

	traceInfo := GetTraceInfo(ctx)
	if traceInfo == nil {
		// given context has no TraceInfo
		// so we can't start new trace to support sampling.
		// closeDummySpan does nohing
		return ctx, closeDummySpan
	}

	var rpcName string
	if len(args) > 0 {
		rpcName = fmt.Sprintf(rpcNameFormat, args...)
	} else {
		rpcName = rpcNameFormat
	}

	// startTime is not used only to log the start of an RPC
	// It's stored in Context to calculate the RPC call duration.
	// A user can get it via Context.Value(TraceStartTimeValue)
	startTime := time.Now()

	// Tracing magic:
	// * the previous span becomes our parent
	// * new span is set as random number
	// * trace still stays the same
	traceInfo.Parent = traceInfo.Span
	traceInfo.Span = uint64(rand.Int63())

	traceInfo.getLog().WithFields(Fields{
		"trace_id":       fmt.Sprintf("%x", traceInfo.Trace),
		"span_id":        fmt.Sprintf("%x", traceInfo.Span),
		"parent_id":      fmt.Sprintf("%x", traceInfo.Parent),
		"real_timestamp": startTime.UnixNano() / 1000,
		"rpc_name":       rpcName,
	}).Infof("start")

	ctx = &traced{
		Context:   ctx,
		traceInfo: *traceInfo,
		startTime: startTime,
	}

	return ctx, func() {
		now := time.Now()
		duration := now.Sub(startTime)
		traceInfo.getLog().WithFields(Fields{
			"trace_id":       fmt.Sprintf("%x", traceInfo.Trace),
			"span_id":        fmt.Sprintf("%x", traceInfo.Span),
			"parent_id":      fmt.Sprintf("%x", traceInfo.Parent),
			"real_timestamp": now.UnixNano() / 1000,
			"duration":       duration.Nanoseconds() / 1000,
			"rpc_name":       rpcName,
		}).Infof("finish")
	}
}
