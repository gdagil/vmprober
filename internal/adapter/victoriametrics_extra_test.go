package adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gdagil/vmprober/internal/config"
	"github.com/gdagil/vmprober/internal/types"
)

// --- test helpers -----------------------------------------------------------

func quietLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

func sampleMetric(name string) types.Metric {
	return types.Metric{
		Name:      name,
		Value:     1,
		Timestamp: time.Now(),
		Type:      types.MetricTypeGauge,
	}
}

// newTestAdapter builds a DefaultVictoriaMetricsAdapter pointed at the given
// endpoints. Retry delays are intentionally large so scheduled retries never
// fire during the short lifetime of a test (they are drained on Stop instead).
func newTestAdapter(t *testing.T, size int, timeout time.Duration, endpoints ...config.EndpointConfig) *DefaultVictoriaMetricsAdapter {
	t.Helper()
	cfg := &config.PushConfig{
		Enabled:   true,
		Endpoints: endpoints,
		Retry: config.RetryConfig{
			MaxAttempts:  3,
			Backoff:      "fixed",
			InitialDelay: 30 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
		},
		Batch: config.BatchConfig{Size: size, Timeout: timeout},
	}
	a, err := NewVictoriaMetricsAdapter(cfg, quietLogger())
	require.NoError(t, err)
	return a.(*DefaultVictoriaMetricsAdapter)
}

// countingServer records how many requests it received and replies with status.
func countingServer(status int) (*httptest.Server, *int64) {
	var count int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(status)
	}))
	return s, &count
}

// waitForCount polls counter until it reaches want or the deadline elapses.
func waitForCount(t *testing.T, counter *int64, want int64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(counter) >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.GreaterOrEqualf(t, atomic.LoadInt64(counter), want,
		"timed out waiting for request count to reach %d", want)
}

// waitQueueDrained waits until the batch queue has been consumed by a worker.
func waitQueueDrained(t *testing.T, a *DefaultVictoriaMetricsAdapter, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(a.batchQueue) == 0 {
			// Give the worker a moment to move the item from the channel
			// receive into its local batch slice.
			time.Sleep(20 * time.Millisecond)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.Fail(t, "batch queue was not drained in time")
}

// errFormatter is a Formatter stub that always fails, to exercise the
// format-error branch of sendBatch.
type errFormatter struct{}

func (errFormatter) Format(_ []types.Metric) ([]byte, error) {
	return nil, errors.New("boom: cannot format")
}

// panicFormatter panics during formatting, to exercise batchWorker's
// panic-recovery guard.
type panicFormatter struct{}

func (panicFormatter) Format(_ []types.Metric) ([]byte, error) {
	panic("formatter exploded")
}

// --- Push -------------------------------------------------------------------

func TestPush_QueueFull_DirectSend(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	// Small pushTimeout so the queue-full fallback path triggers quickly.
	a := newTestAdapter(t, 1, 80*time.Millisecond, config.EndpointConfig{URL: server.URL})
	defer a.Stop(context.Background())

	// Fill the queue (capacity == Batch.Size == 1) so Push cannot enqueue.
	a.batchQueue <- []types.Metric{sampleMetric("filler")}

	// No worker is running, so the send blocks until pushTimeout, then Push
	// falls back to sending the batch directly to the endpoint.
	err := a.Push(context.Background(), []types.Metric{sampleMetric("direct")})
	require.NoError(t, err)

	waitForCount(t, count, 1, 2*time.Second)
	assert.EqualValues(t, 1, atomic.LoadInt64(count))
}

func TestPush_ParentContextCanceled(t *testing.T) {
	server, _ := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 1, 5*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.Stop(context.Background())

	// Fill the queue so the enqueue case blocks.
	a.batchQueue <- []types.Metric{sampleMetric("filler")}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled

	err := a.Push(ctx, []types.Metric{sampleMetric("m")})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPush_AdapterStopped(t *testing.T) {
	server, _ := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 1, 5*time.Second, config.EndpointConfig{URL: server.URL})

	// Fill the queue so the enqueue case blocks, then cancel the adapter's
	// internal context so Push takes the "adapter is stopped" path.
	a.batchQueue <- []types.Metric{sampleMetric("filler")}
	a.cancel()

	err := a.Push(context.Background(), []types.Metric{sampleMetric("m")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "adapter is stopped")

	// Clean up the retry engine goroutine (queue already holds the filler, so
	// avoid the double-close that a full Stop would perform after cancel()).
	a.retryEngine.Stop()
}

// --- batchWorker ------------------------------------------------------------

func TestBatchWorker_SizeFlush(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 3, 10*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.wg.Add(1)
	go a.batchWorker(ctx)

	// One push of exactly Batch.Size metrics triggers an immediate size flush.
	a.batchQueue <- []types.Metric{sampleMetric("a"), sampleMetric("b"), sampleMetric("c")}

	waitForCount(t, count, 1, 2*time.Second)
	cancel()
	a.wg.Wait()

	assert.EqualValues(t, 1, atomic.LoadInt64(count))
	assert.EqualValues(t, 3, a.GetStats().TotalPushed)
}

func TestBatchWorker_TickerFlush(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	// Large size, small timeout: the batch is flushed by the ticker, not size.
	a := newTestAdapter(t, 100, 40*time.Millisecond, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.wg.Add(1)
	go a.batchWorker(ctx)

	a.batchQueue <- []types.Metric{sampleMetric("tick")}

	waitForCount(t, count, 1, 2*time.Second)
	cancel()
	a.wg.Wait()

	assert.GreaterOrEqual(t, atomic.LoadInt64(count), int64(1))
	assert.EqualValues(t, 1, a.GetStats().TotalPushed)
}

func TestBatchWorker_ChannelCloseDrains(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 10*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a.wg.Add(1)
	go a.batchWorker(ctx)

	// Below the size threshold, so it accumulates; closing the channel forces
	// the worker to drain the remaining batch before returning.
	a.batchQueue <- []types.Metric{sampleMetric("pending")}
	close(a.batchQueue)

	a.wg.Wait()

	waitForCount(t, count, 1, 2*time.Second)
	assert.EqualValues(t, 1, a.GetStats().TotalPushed)
}

func TestBatchWorker_AdapterCtxDoneDrains(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 10*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()

	// Parent ctx stays alive; the adapter's own ctx is what we cancel.
	a.wg.Add(1)
	go a.batchWorker(context.Background())

	a.batchQueue <- []types.Metric{sampleMetric("pending")}
	waitQueueDrained(t, a, 2*time.Second)

	a.cancel() // triggers the a.ctx.Done drain branch
	a.wg.Wait()

	waitForCount(t, count, 1, 2*time.Second)
	assert.EqualValues(t, 1, a.GetStats().TotalPushed)
}

func TestBatchWorker_ParentCtxDoneDrains(t *testing.T) {
	// The parent ctx is canceled while a batch is pending. The drain send uses
	// that canceled context, so the HTTP request fails and the batch is counted
	// as failed - exercising the ctx.Done drain path and the failure branch.
	server, _ := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 10*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	a.wg.Add(1)
	go a.batchWorker(ctx)

	a.batchQueue <- []types.Metric{sampleMetric("pending")}
	waitQueueDrained(t, a, 2*time.Second)

	cancel() // triggers the parent ctx.Done drain branch with a canceled ctx
	a.wg.Wait()

	assert.EqualValues(t, 1, a.GetStats().TotalFailed)
}

func TestBatchWorker_RecoversFromPanic(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 1, 10*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.retryEngine.Stop()
	a.formatter = panicFormatter{}

	a.wg.Add(1)
	go a.batchWorker(context.Background())

	// A size-triggering push makes the worker call sendBatch, whose formatter
	// panics; the deferred recover must keep the goroutine from crashing and
	// let it return cleanly (so wg.Wait unblocks).
	a.batchQueue <- []types.Metric{sampleMetric("boom")}

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("batchWorker did not recover from panic and exit")
	}

	assert.EqualValues(t, 0, atomic.LoadInt64(count)) // panic happened before send
}

// --- sendBatch --------------------------------------------------------------

func TestSendBatch_Empty(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.Stop(context.Background())

	require.NoError(t, a.sendBatch(context.Background(), nil))
	assert.EqualValues(t, 0, atomic.LoadInt64(count))
	assert.EqualValues(t, 0, a.GetStats().TotalPushed)
}

func TestSendBatch_Success_UpdatesStatsAndSuccessRate(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.Stop(context.Background())

	err := a.sendBatch(context.Background(), []types.Metric{sampleMetric("a"), sampleMetric("b")})
	require.NoError(t, err)

	assert.EqualValues(t, 1, atomic.LoadInt64(count))

	stats := a.GetStats()
	assert.EqualValues(t, 2, stats.TotalPushed)
	assert.EqualValues(t, 0, stats.TotalFailed)
	assert.Equal(t, 100.0, stats.SuccessRate)
	assert.False(t, stats.LastPushTime.IsZero())
}

func TestSendBatch_ServerError_SchedulesRetryAndFails(t *testing.T) {
	server, count := countingServer(http.StatusInternalServerError)
	defer server.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: server.URL})

	err := a.sendBatch(context.Background(), []types.Metric{sampleMetric("a")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")

	// One initial delivery attempt was made; the retry is scheduled with a long
	// delay and dropped on Stop, so it does not hit the server again here.
	assert.EqualValues(t, 1, atomic.LoadInt64(count))

	stats := a.GetStats()
	assert.EqualValues(t, 1, stats.TotalFailed)
	assert.EqualValues(t, 0, stats.TotalPushed)
	assert.Equal(t, 0.0, stats.SuccessRate)

	// A retry task should have been queued (ShouldRetry always true here).
	a.retryEngine.mu.RLock()
	scheduled := a.retryEngine.stats.TotalRetries
	a.retryEngine.mu.RUnlock()
	assert.EqualValues(t, 1, scheduled)

	require.NoError(t, a.Stop(context.Background()))
}

func TestSendBatch_FormatError(t *testing.T) {
	server, count := countingServer(http.StatusOK)
	defer server.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: server.URL})
	defer a.Stop(context.Background())
	a.formatter = errFormatter{}

	err := a.sendBatch(context.Background(), []types.Metric{sampleMetric("a")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to format metrics")
	assert.EqualValues(t, 0, atomic.LoadInt64(count)) // never reached the endpoint
}

func TestSendBatch_MultiEndpoint_PartialSuccess(t *testing.T) {
	good, goodCount := countingServer(http.StatusOK)
	defer good.Close()
	bad, badCount := countingServer(http.StatusBadGateway)
	defer bad.Close()

	// The good endpoint uses bearer auth so we also assert header injection.
	var sawBearer int64
	goodAuth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer secret-token" {
			atomic.AddInt64(&sawBearer, 1)
		}
		atomic.AddInt64(goodCount, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer goodAuth.Close()

	a := newTestAdapter(t, 100, 5*time.Second,
		config.EndpointConfig{URL: bad.URL},
		config.EndpointConfig{
			URL:  goodAuth.URL,
			Auth: config.AuthConfig{Type: "bearer", Token: "secret-token"},
		},
	)
	defer a.Stop(context.Background())

	// One endpoint fails, one succeeds -> overall success (lastErr cleared).
	err := a.sendBatch(context.Background(), []types.Metric{sampleMetric("a")})
	require.NoError(t, err)

	assert.EqualValues(t, 1, atomic.LoadInt64(badCount))
	assert.EqualValues(t, 1, atomic.LoadInt64(goodCount))
	assert.EqualValues(t, 1, atomic.LoadInt64(&sawBearer))
	assert.EqualValues(t, 1, a.GetStats().TotalPushed)
}

// --- sendToEndpoint edge cases ---------------------------------------------

func TestSendToEndpoint_BasicAuthInjection(t *testing.T) {
	var okAuth int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); ok && u == "user" && p == "pass" {
			atomic.AddInt64(&okAuth, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{
		URL:  server.URL,
		Auth: config.AuthConfig{Type: "basic", Username: "user", Password: "pass"},
	})
	defer a.Stop(context.Background())

	err := a.sendToEndpoint(context.Background(), a.endpoints[0], []byte("data"))
	require.NoError(t, err)
	assert.EqualValues(t, 1, atomic.LoadInt64(&okAuth))
}

func TestSendToEndpoint_RequestError(t *testing.T) {
	// Point at a closed listener address so the HTTP request fails to connect,
	// exercising the client.Do error branch.
	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: "http://127.0.0.1:1"})
	defer a.Stop(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := a.sendToEndpoint(ctx, a.endpoints[0], []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestSendToEndpoint_BadURL(t *testing.T) {
	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: "http://example.com"})
	defer a.Stop(context.Background())

	bad := &Endpoint{URL: "http://foo\x7f.com/"} // control char -> request build fails
	err := a.sendToEndpoint(context.Background(), bad, []byte("data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

// --- GetStats ---------------------------------------------------------------

func TestGetStats_SuccessRateMixed(t *testing.T) {
	good, _ := countingServer(http.StatusOK)
	defer good.Close()
	bad, _ := countingServer(http.StatusInternalServerError)
	defer bad.Close()

	a := newTestAdapter(t, 100, 5*time.Second, config.EndpointConfig{URL: good.URL})
	defer a.Stop(context.Background())

	// One successful batch of 3.
	require.NoError(t, a.sendBatch(context.Background(), []types.Metric{
		sampleMetric("a"), sampleMetric("b"), sampleMetric("c"),
	}))

	// Repoint at the failing endpoint for one failed batch of 1.
	a.endpoints[0].URL = bad.URL
	require.Error(t, a.sendBatch(context.Background(), []types.Metric{sampleMetric("d")}))

	stats := a.GetStats()
	assert.EqualValues(t, 3, stats.TotalPushed)
	assert.EqualValues(t, 1, stats.TotalFailed)
	assert.InDelta(t, 75.0, stats.SuccessRate, 0.001) // 3 / (3+1) * 100
}

// --- ScheduleRetry queue-full ----------------------------------------------

func TestScheduleRetry_QueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build a RetryEngine directly with a full, undrained queue (no worker) so
	// ScheduleRetry hits the default "queue full, dropping" branch.
	r := &RetryEngine{
		config:     createTestRetryConfig(),
		logger:     quietLogger(),
		retryQueue: make(chan retryTask, 1),
		stats:      &RetryStats{},
		ctx:        ctx,
		cancel:     cancel,
	}
	r.retryQueue <- retryTask{endpoint: "prefilled"} // queue now full

	before := r.stats.TotalRetries
	r.ScheduleRetry(context.Background(), "http://full", func() error { return nil })

	// Dropped task must not be counted as scheduled.
	assert.Equal(t, before, r.stats.TotalRetries)
	assert.Len(t, r.retryQueue, 1)
}
