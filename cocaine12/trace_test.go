package cocaine12

import (
	"testing"
)

func BenchmarkTraceWith(b *testing.B) {
	ctx := BeginNewTraceContext()
	for n := 0; n < b.N; n++ {
		_, _ = WithTrace(ctx, "bench")
	}
}
