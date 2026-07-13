package metrics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gdagil/vmprober/internal/types"
)

// TestExportMetricsNoDeadlock reproduces the RWMutex deadlock fixed in
// ExportMetrics: WritePrometheus runs the job gauge callbacks (RLock) while a
// writer (UpdateJobMetrics/Record) queues on the same mutex. With the bug the
// goroutines lock up and the watchdog fails the test; with the fix the hammer
// completes quickly.
func TestExportMetricsNoDeadlock(t *testing.T) {
	collector := NewCollector("vmprober_dltest", true, map[string]string{"job": "test"})
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		stop := time.Now().Add(2 * time.Second)
		for w := 0; w < 4; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for time.Now().Before(stop) {
					switch w % 3 {
					case 0:
						collector.ExportMetrics()
					case 1:
						collector.UpdateJobMetrics(10, 2, 1)
					default:
						_ = collector.Record(ctx, &types.ProbeResult{
							TargetHost: "host", TargetIP: "1.2.3.4", TargetPort: 80,
							Protocol: types.ProbeTypeTCP, Success: true, RTT: time.Millisecond,
						})
					}
				}
			}(w)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock: ExportMetrics/UpdateJobMetrics/Record did not finish in 10s")
	}
}
