package cocaine12

import (
	"testing"
)

func BenchmarkTraceWith(b *testing.B) {
	ctx := BeginNewTraceContext(nil)
	for n := 0; n < b.N; n++ {
		_, _ = WithTrace(ctx, "bench")
	}
}
