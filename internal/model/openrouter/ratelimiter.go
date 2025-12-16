// Package openrouter provides an OpenRouter model implementation for Google ADK.
package openrouter

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	// DefaultMaxConcurrent is the default maximum concurrent requests to OpenRouter
	DefaultMaxConcurrent = 5
	// DefaultMinDelay is the minimum delay between requests (helps respect spend caps)
	DefaultMinDelay = 100 * time.Millisecond
)

// RateLimiter provides global rate limiting for OpenRouter API calls.
// It uses a semaphore to limit concurrent requests and an optional delay
// between requests to help respect spend caps.
type RateLimiter struct {
	semaphore     chan struct{}
	maxConcurrent int
	minDelay      time.Duration
	lastRequest   time.Time
	mu            sync.Mutex
}

// Global rate limiter instance (singleton)
var (
	globalRateLimiter *RateLimiter
	rateLimiterOnce   sync.Once
)

// GetGlobalRateLimiter returns the global rate limiter instance.
// Configuration can be set via environment variables:
// - OPENROUTER_MAX_CONCURRENT: maximum concurrent requests (default: 5)
// - OPENROUTER_MIN_DELAY_MS: minimum delay between requests in ms (default: 100)
func GetGlobalRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		maxConcurrent := DefaultMaxConcurrent
		if val := os.Getenv("OPENROUTER_MAX_CONCURRENT"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				maxConcurrent = n
			}
		}

		minDelay := DefaultMinDelay
		if val := os.Getenv("OPENROUTER_MIN_DELAY_MS"); val != "" {
			if n, err := strconv.Atoi(val); err == nil && n >= 0 {
				minDelay = time.Duration(n) * time.Millisecond
			}
		}

		globalRateLimiter = &RateLimiter{
			semaphore:     make(chan struct{}, maxConcurrent),
			maxConcurrent: maxConcurrent,
			minDelay:      minDelay,
		}

		log.Printf("[OpenRouter RateLimiter] Initialized: max_concurrent=%d, min_delay=%v",
			maxConcurrent, minDelay)
	})

	return globalRateLimiter
}

// Acquire acquires a slot in the rate limiter.
// It blocks until a slot is available or the context is cancelled.
// Returns a release function that MUST be called when the request is complete.
func (r *RateLimiter) Acquire(ctx context.Context) (release func(), err error) {
	// Wait for semaphore slot
	select {
	case r.semaphore <- struct{}{}:
		// Got a slot
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Apply minimum delay between requests
	r.mu.Lock()
	if r.minDelay > 0 {
		elapsed := time.Since(r.lastRequest)
		if elapsed < r.minDelay {
			sleepTime := r.minDelay - elapsed
			r.mu.Unlock()

			select {
			case <-time.After(sleepTime):
			case <-ctx.Done():
				// Release semaphore on cancellation
				<-r.semaphore
				return nil, ctx.Err()
			}

			r.mu.Lock()
		}
	}
	r.lastRequest = time.Now()
	r.mu.Unlock()

	// Return release function
	return func() {
		<-r.semaphore
	}, nil
}

// TryAcquire tries to acquire a slot without blocking.
// Returns false if no slot is available.
func (r *RateLimiter) TryAcquire() (release func(), ok bool) {
	select {
	case r.semaphore <- struct{}{}:
		return func() { <-r.semaphore }, true
	default:
		return nil, false
	}
}

// CurrentUsage returns the number of slots currently in use.
func (r *RateLimiter) CurrentUsage() int {
	return len(r.semaphore)
}

// MaxConcurrent returns the maximum concurrent requests allowed.
func (r *RateLimiter) MaxConcurrent() int {
	return r.maxConcurrent
}
