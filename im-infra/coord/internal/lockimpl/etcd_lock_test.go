package lockimpl_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/internal/lockimpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Setup ---

var (
	etcdEndpoints = []string{"localhost:2379"}
)

// helper function to create a new lock factory for testing
func newTestLockFactory(t *testing.T) *lockimpl.EtcdLockFactory {
	t.Helper()

	// Create a logger for testing
	logCfg := clog.DefaultConfig()
	logCfg.Level = "debug" // Enable debug level for detailed test output
	testLogger, err := clog.New(logCfg)
	require.NoError(t, err, "Failed to create logger for testing")

	cli, err := client.New(client.Config{
		Endpoints: etcdEndpoints,
		Timeout:   5 * time.Second,
		Logger:    testLogger.With(clog.String("component", "etcd-client-test")),
	})
	require.NoError(t, err, "Failed to connect to etcd for testing")

	// Clean up etcd client after test suite finishes
	t.Cleanup(func() {
		assert.NoError(t, cli.Close())
	})

	return lockimpl.NewEtcdLockFactory(cli, "/test-locks", testLogger)
}

// --- Test Cases ---

// TestBasicAcquireAndUnlock tests the fundamental lock and unlock functionality.
func TestBasicAcquireAndUnlock(t *testing.T) {
	factory := newTestLockFactory(t)
	lockKey := "basic-test"
	ctx := context.Background()

	// 1. Acquire the lock
	l, err := factory.Acquire(ctx, lockKey, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, l)
	t.Logf("Lock acquired: %s", l.Key())

	// 2. Check TTL
	ttl, err := l.TTL(ctx)
	require.NoError(t, err)
	assert.True(t, ttl > 0 && ttl <= 5*time.Second, "TTL should be positive and less than or equal to 5s")
	t.Logf("Initial TTL: %v", ttl)

	// 3. Unlock the lock
	err = l.Unlock(ctx)
	require.NoError(t, err)
	t.Log("Lock unlocked successfully")

	// 4. Verify lock can be acquired again
	l2, err := factory.TryAcquire(ctx, lockKey, 5*time.Second)
	require.NoError(t, err, "Should be able to acquire lock again after unlock")
	require.NotNil(t, l2)
	defer l2.Unlock(ctx)
	t.Log("Lock re-acquired successfully")
}

// TestTryAcquire tests the non-blocking lock acquisition.
func TestTryAcquire(t *testing.T) {
	factory := newTestLockFactory(t)
	lockKey := "try-acquire-test"
	ctx := context.Background()

	// 1. Acquire a lock to hold it
	l1, err := factory.Acquire(ctx, lockKey, 10*time.Second)
	require.NoError(t, err)
	defer l1.Unlock(ctx)
	t.Log("Worker 1 acquired lock")

	// 2. Try to acquire the same lock, expect it to fail immediately
	l2, err := factory.TryAcquire(ctx, lockKey, 10*time.Second)
	assert.Error(t, err, "TryAcquire should fail when lock is held")
	assert.Nil(t, l2, "Returned lock should be nil on failure")

	// Check for the specific conflict error
	clientErr, ok := err.(*client.Error)
	require.True(t, ok, "Error should be of type *client.Error")
	assert.Equal(t, client.ErrCodeConflict, clientErr.Code, "Error code should be conflict")
	t.Log("Worker 2 failed to TryAcquire as expected")
}

// TestLockExpiration simulates a client crash and checks if the lock is released.
func TestLockExpiration(t *testing.T) {
	lockKey := "expiration-test"
	ctx := context.Background()
	leaseTTL := 3 * time.Second // Use a short TTL for the test

	// --- Client 1: Acquires the lock and then "crashes" ---
	logCfg := clog.DefaultConfig()
	logCfg.Level = "debug"
	testLogger, err := clog.New(logCfg)
	require.NoError(t, err)

	client1, err := client.New(client.Config{
		Endpoints: etcdEndpoints,
		Timeout:   5 * time.Second,
		Logger:    testLogger.With(clog.String("component", "etcd-client-1")),
	})
	require.NoError(t, err)
	factory1 := lockimpl.NewEtcdLockFactory(client1, "/test-locks", testLogger)

	// Client 1 acquires the lock with a short TTL
	l1, err := factory1.Acquire(ctx, lockKey, leaseTTL)
	require.NoError(t, err)
	require.NotNil(t, l1)
	t.Log("Client 1 acquired lock")

	// Get the lease ID so we can revoke it manually to simulate crash
	ttl, err := l1.TTL(ctx)
	require.NoError(t, err)
	t.Logf("Lock acquired with TTL: %v", ttl)

	// Simulate crash by closing the client connection immediately
	// This prevents keepalive from renewing the lease
	err = client1.Close()
	require.NoError(t, err)
	t.Log("Client 1 connection closed, simulating crash")

	// Wait for lease to naturally expire (TTL + some buffer)
	// Since keepalive stopped, the lease should expire after its TTL
	waitDuration := leaseTTL + 2*time.Second
	t.Logf("Waiting for lease to expire naturally (%v)...", waitDuration)
	time.Sleep(waitDuration)

	// --- Client 2: Tries to acquire the lock ---
	t.Log("Client 2 starting...")
	factory2 := newTestLockFactory(t)

	// Set a reasonable timeout for Client 2's acquisition attempt
	acquireCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// After the lease expires, client 2 should be able to acquire the lock.
	l2, err := factory2.Acquire(acquireCtx, lockKey, 5*time.Second)
	require.NoError(t, err, "Client 2 should acquire lock after Client 1's lease expires")
	require.NotNil(t, l2)
	t.Log("Client 2 successfully acquired the lock after Client 1 crashed.")

	// Clean up
	err = l2.Unlock(ctx)
	require.NoError(t, err)
}

// TestAutoRenewal verifies that the lock's lease is automatically renewed.
func TestAutoRenewal(t *testing.T) {
	factory := newTestLockFactory(t)
	lockKey := "auto-renewal-test"
	ctx := context.Background()
	ttl := 3 * time.Second // Short TTL

	// 1. Acquire lock
	l, err := factory.Acquire(ctx, lockKey, ttl)
	require.NoError(t, err)
	t.Logf("Lock acquired with TTL %v", ttl)

	// 2. Wait for a period longer than the initial TTL
	waitDuration := ttl + 2*time.Second
	t.Logf("Holding lock for %v (longer than TTL) to test auto-renewal...", waitDuration)
	time.Sleep(waitDuration)

	// 3. Check if another client can acquire the lock (it shouldn't)
	_, err = factory.TryAcquire(ctx, lockKey, ttl)
	assert.Error(t, err, "Lock should still be held due to auto-renewal")
	t.Log("TryAcquire failed as expected, proving lock is still held")

	// 4. Check the remaining TTL of the original lock
	remainingTTL, err := l.TTL(ctx)
	require.NoError(t, err)
	assert.True(t, remainingTTL > 0, "Lock should still have time remaining")
	t.Logf("Remaining TTL after auto-renewal: %v", remainingTTL)

	// 5. Clean up
	err = l.Unlock(ctx)
	require.NoError(t, err)
}

// TestContextCancellation verifies that a blocking Acquire call is cancelled by context.
func TestContextCancellation(t *testing.T) {
	factory := newTestLockFactory(t)
	lockKey := "cancellation-test"
	ctx := context.Background()

	// 1. Acquire and hold the lock
	l1, err := factory.Acquire(ctx, lockKey, 10*time.Second)
	require.NoError(t, err)
	defer l1.Unlock(ctx)
	t.Log("Worker 1 acquired lock")

	// 2. Start a goroutine to acquire the same lock, but with a cancellable context
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Log("Worker 2 attempting to acquire lock with a 2s timeout...")
		ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		_, err := factory.Acquire(ctx2, lockKey, 10*time.Second)
		assert.Error(t, err, "Acquire should be cancelled by context timeout")
		// Check if the error is context.DeadlineExceeded
		assert.Equal(t, context.DeadlineExceeded, err.(*client.Error).Unwrap(), "Error should be context.DeadlineExceeded")
		t.Log("Worker 2's Acquire call failed with context deadline exceeded, as expected")
	}()

	wg.Wait()
}

// TestHighConcurrencyCompetition tests the mutex under high contention.
func TestHighConcurrencyCompetition(t *testing.T) {
	factory := newTestLockFactory(t)
	lockKey := "concurrency-test"
	numGoroutines := 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// A shared variable to be protected by the lock.
	// We use a map to check if a worker ran at the same time as another.
	criticalSectionTracker := make(map[int]bool)
	var mu sync.Mutex // Mutex for the tracker map itself
	var acquisitions int32

	// Use a channel to make goroutines start at roughly the same time
	startCh := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			<-startCh // Wait for the signal to start

			ctx := context.Background()
			l, err := factory.Acquire(ctx, lockKey, 5*time.Second)
			if err != nil {
				t.Errorf("Goroutine %d failed to acquire lock: %v", id, err)
				return
			}

			// --- Critical Section ---
			mu.Lock()
			if len(criticalSectionTracker) > 0 {
				t.Errorf("Mutex failed! Goroutine %d entered critical section while others were present: %v", id, criticalSectionTracker)
			}
			criticalSectionTracker[id] = true
			acquisitions++
			mu.Unlock()

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			delete(criticalSectionTracker, id)
			mu.Unlock()
			// --- End Critical Section ---

			if err := l.Unlock(ctx); err != nil {
				t.Errorf("Goroutine %d failed to unlock: %v", id, err)
			}
		}(i)
	}

	t.Logf("Starting %d goroutines to compete for the lock...", numGoroutines)
	close(startCh) // Signal all goroutines to start
	wg.Wait()

	assert.Equal(t, int32(numGoroutines), acquisitions, fmt.Sprintf("Expected %d acquisitions, but got %d", numGoroutines, acquisitions))
	t.Log("High concurrency test finished. All goroutines acquired and released the lock sequentially.")
}
