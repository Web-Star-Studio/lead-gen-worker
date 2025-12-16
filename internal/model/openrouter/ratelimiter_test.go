package openrouter

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_Acquire(t *testing.T) {
	// Create a fresh rate limiter for testing
	rl := &RateLimiter{
		semaphore:     make(chan struct{}, 3),
		maxConcurrent: 3,
		minDelay:      0, // No delay for faster tests
	}

	ctx := context.Background()

	t.Run("acquires and releases slots", func(t *testing.T) {
		release1, err := rl.Acquire(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, rl.CurrentUsage())

		release2, err := rl.Acquire(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, rl.CurrentUsage())

		release1()
		assert.Equal(t, 1, rl.CurrentUsage())

		release2()
		assert.Equal(t, 0, rl.CurrentUsage())
	})

	t.Run("respects max concurrent limit", func(t *testing.T) {
		// Acquire all slots
		var releases []func()
		for i := 0; i < 3; i++ {
			release, err := rl.Acquire(ctx)
			require.NoError(t, err)
			releases = append(releases, release)
		}
		assert.Equal(t, 3, rl.CurrentUsage())

		// Try to acquire one more with timeout
		ctxTimeout, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		_, err := rl.Acquire(ctxTimeout)
		assert.Error(t, err) // Should timeout

		// Release all
		for _, release := range releases {
			release()
		}
		assert.Equal(t, 0, rl.CurrentUsage())
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Fill up the semaphore
		var releases []func()
		for i := 0; i < 3; i++ {
			release, err := rl.Acquire(ctx)
			require.NoError(t, err)
			releases = append(releases, release)
		}

		// Try to acquire with cancelled context
		ctxCancelled, cancel := context.WithCancel(ctx)
		cancel()

		_, err := rl.Acquire(ctxCancelled)
		assert.Error(t, err)

		// Cleanup
		for _, release := range releases {
			release()
		}
	})
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl := &RateLimiter{
		semaphore:     make(chan struct{}, 5),
		maxConcurrent: 5,
		minDelay:      0,
	}

	ctx := context.Background()
	var maxConcurrent int32
	var currentConcurrent int32
	var wg sync.WaitGroup

	// Launch 20 goroutines that try to acquire the rate limiter
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			release, err := rl.Acquire(ctx)
			if err != nil {
				return
			}
			defer release()

			// Track concurrent usage
			current := atomic.AddInt32(&currentConcurrent, 1)
			for {
				old := atomic.LoadInt32(&maxConcurrent)
				if current <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
					break
				}
			}

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			atomic.AddInt32(&currentConcurrent, -1)
		}()
	}

	wg.Wait()

	// Max concurrent should not exceed 5
	assert.LessOrEqual(t, int(maxConcurrent), 5, "Max concurrent should not exceed limit")
	assert.Equal(t, 0, rl.CurrentUsage(), "All slots should be released")
}

func TestRateLimiter_TryAcquire(t *testing.T) {
	rl := &RateLimiter{
		semaphore:     make(chan struct{}, 2),
		maxConcurrent: 2,
		minDelay:      0,
	}

	// Should succeed
	release1, ok := rl.TryAcquire()
	assert.True(t, ok)
	assert.Equal(t, 1, rl.CurrentUsage())

	release2, ok := rl.TryAcquire()
	assert.True(t, ok)
	assert.Equal(t, 2, rl.CurrentUsage())

	// Should fail (non-blocking)
	_, ok = rl.TryAcquire()
	assert.False(t, ok)

	// Release and try again
	release1()
	release3, ok := rl.TryAcquire()
	assert.True(t, ok)

	release2()
	release3()
	assert.Equal(t, 0, rl.CurrentUsage())
}

func TestRateLimiter_MinDelay(t *testing.T) {
	rl := &RateLimiter{
		semaphore:     make(chan struct{}, 10),
		maxConcurrent: 10,
		minDelay:      50 * time.Millisecond,
	}

	ctx := context.Background()

	// First request should be immediate
	start := time.Now()
	release1, err := rl.Acquire(ctx)
	require.NoError(t, err)
	release1()
	firstDuration := time.Since(start)

	// Second request should be delayed
	start = time.Now()
	release2, err := rl.Acquire(ctx)
	require.NoError(t, err)
	release2()
	secondDuration := time.Since(start)

	// Second request should have taken at least minDelay
	assert.Less(t, firstDuration, 20*time.Millisecond, "First request should be fast")
	assert.GreaterOrEqual(t, secondDuration, 40*time.Millisecond, "Second request should be delayed")
}
