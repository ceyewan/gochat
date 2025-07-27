package coord_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) coord.Provider {
	provider, err := coord.New(context.Background())
	require.NoError(t, err, "Failed to create coordinator")
	t.Cleanup(func() {
		err := provider.Close()
		assert.NoError(t, err, "Failed to close provider")
	})
	return provider
}

func TestLock(t *testing.T) {
	provider := setup(t)
	lockService := provider.Lock()
	lockKey := "test-lock"

	t.Run("AcquireAndRelease", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		l, err := lockService.Acquire(ctx, lockKey, 10*time.Second)
		require.NoError(t, err, "Should acquire lock without error")
		assert.Contains(t, l.Key(), lockKey, "The returned key should contain the original lock key")

		err = l.Unlock(ctx)
		assert.NoError(t, err, "Should unlock without error")
	})

	t.Run("TryAcquireSuccess", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		l, err := lockService.TryAcquire(ctx, lockKey, 10*time.Second)
		require.NoError(t, err, "Should try acquire lock without error")
		defer func() {
			err := l.Unlock(context.Background())
			assert.NoError(t, err)
		}()
		assert.Contains(t, l.Key(), lockKey, "The returned key should contain the original lock key")
	})

	t.Run("TryAcquireFailWhenLocked", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// First, acquire the lock
		l1, err := lockService.Acquire(ctx, lockKey, 10*time.Second)
		require.NoError(t, err)
		defer func() {
			err := l1.Unlock(context.Background())
			assert.NoError(t, err)
		}()

		// Then, try to acquire it again, should fail
		_, err = lockService.TryAcquire(ctx, lockKey, 10*time.Second)
		assert.Error(t, err, "Should fail to acquire a locked lock")
	})

	t.Run("LockCompetition", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		var firstAcquired, secondAcquired time.Time

		// Goroutine 1
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			l, err := lockService.Acquire(ctx, lockKey, 5*time.Second)
			if err == nil {
				firstAcquired = time.Now()
				time.Sleep(1 * time.Second)
				_ = l.Unlock(ctx)
			}
		}()

		// Goroutine 2
		go func() {
			defer wg.Done()
			time.Sleep(100 * time.Millisecond) // Ensure goroutine 1 starts first
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			l, err := lockService.Acquire(ctx, lockKey, 5*time.Second)
			if err == nil {
				secondAcquired = time.Now()
				_ = l.Unlock(ctx)
			}
		}()

		wg.Wait()
		assert.True(t, secondAcquired.After(firstAcquired), "Second goroutine should acquire lock after the first one")
	})

	t.Run("LockExpires", func(t *testing.T) {
		// Create a separate provider to simulate a client that will "crash"
		expiringProvider := setup(t)
		expiringLockService := expiringProvider.Lock()

		// Acquire a lock with a short TTL
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := expiringLockService.Acquire(ctx, lockKey, 2*time.Second)
		require.NoError(t, err)

		// "Crash" the client by closing the provider, so the lease won't be renewed
		err = expiringProvider.Close()
		require.NoError(t, err)

		// Wait for the lease to expire
		time.Sleep(3 * time.Second)

		// Now, the original provider should be able to acquire the lock
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		l2, err := lockService.TryAcquire(ctx2, lockKey, 5*time.Second)
		require.NoError(t, err, "Should be able to acquire an expired lock")
		defer func() {
			_ = l2.Unlock(context.Background())
		}()
	})
}

func TestRegistry(t *testing.T) {
	provider := setup(t)
	registryService := provider.Registry()
	serviceName := "test-service"

	serviceInfo1 := registry.ServiceInfo{
		ID:      "test-service-1",
		Name:    serviceName,
		Address: "localhost",
		Port:    8080,
	}
	serviceInfo2 := registry.ServiceInfo{
		ID:      "test-service-2",
		Name:    serviceName,
		Address: "localhost",
		Port:    8081,
	}

	t.Run("RegisterAndDiscover", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := registryService.Register(ctx, serviceInfo1, 10*time.Second)
		require.NoError(t, err, "Should register service without error")
		defer func() {
			_ = registryService.Unregister(context.Background(), serviceInfo1.ID)
		}()

		services, err := registryService.Discover(ctx, serviceName)
		require.NoError(t, err, "Should discover service without error")
		assert.Len(t, services, 1, "Should discover one service")
		assert.Equal(t, serviceInfo1.ID, services[0].ID)
	})

	t.Run("DiscoverNonExistent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		services, err := registryService.Discover(ctx, "non-existent-service")
		require.NoError(t, err, "Should not return error for non-existent service")
		assert.Len(t, services, 0, "Should return no services")
	})

	t.Run("Unregister", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := registryService.Register(ctx, serviceInfo1, 10*time.Second)
		require.NoError(t, err)

		err = registryService.Unregister(ctx, serviceInfo1.ID)
		require.NoError(t, err, "Should unregister service without error")

		services, err := registryService.Discover(ctx, serviceName)
		require.NoError(t, err)
		assert.Len(t, services, 0, "Should discover no services after unregister")
	})

	t.Run("WatchEvents", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		watchCh, err := registryService.Watch(ctx, serviceName)
		require.NoError(t, err, "Should start watch without error")

		var wg sync.WaitGroup
		wg.Add(2) // Expecting two events: PUT and DELETE

		go func() {
			// Register a service to trigger a PUT event
			time.Sleep(100 * time.Millisecond)
			err := registryService.Register(ctx, serviceInfo2, 5*time.Second)
			assert.NoError(t, err)

			// Unregister to trigger a DELETE event
			time.Sleep(100 * time.Millisecond)
			err = registryService.Unregister(ctx, serviceInfo2.ID)
			assert.NoError(t, err)
		}()

		go func() {
			eventCount := 0
			for event := range watchCh {
				if event.Service.ID == serviceInfo2.ID {
					eventCount++
					if event.Type == registry.EventTypePut {
						wg.Done()
					}
					if event.Type == registry.EventTypeDelete {
						wg.Done()
					}
				}
				if eventCount == 2 {
					break
				}
			}
		}()

		wg.Wait()
	})

	t.Run("LeaseExpiration", func(t *testing.T) {
		// Use a separate provider to register a service that will "crash"
		expiringProvider := setup(t)
		expiringRegistry := expiringProvider.Registry()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Register with a short lease
		err := expiringRegistry.Register(ctx, serviceInfo1, 2*time.Second)
		require.NoError(t, err)

		// "Crash" the client by closing the provider
		err = expiringProvider.Close()
		require.NoError(t, err)

		// Wait for the lease to expire
		time.Sleep(3 * time.Second)

		// Now, the original registry should not find the service
		services, err := registryService.Discover(context.Background(), serviceName)
		require.NoError(t, err)
		assert.Len(t, services, 0, "Service should be gone after lease expires")
	})
}

type AppConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func TestConfig(t *testing.T) {
	provider := setup(t)
	configCenter := provider.Config()
	key := "test-config-key"
	prefix := "test-config-prefix"

	t.Run("SetAndGet", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		configValue := &AppConfig{Name: "test-app", Version: "1.0.0"}
		err := configCenter.Set(ctx, key, configValue)
		require.NoError(t, err, "Should set config without error")
		defer func() {
			_ = configCenter.Delete(context.Background(), key)
		}()

		var retrievedValue AppConfig
		err = configCenter.Get(ctx, key, &retrievedValue)
		require.NoError(t, err, "Should get config without error")
		assert.Equal(t, configValue.Name, retrievedValue.Name)
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var value AppConfig
		err := configCenter.Get(ctx, "non-existent-key", &value)
		assert.Error(t, err, "Should return error for non-existent key")
	})

	t.Run("Delete", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := configCenter.Set(ctx, key, "value")
		require.NoError(t, err)

		err = configCenter.Delete(ctx, key)
		require.NoError(t, err, "Should delete config without error")

		var value string
		err = configCenter.Get(ctx, key, &value)
		assert.Error(t, err, "Should return error after deleting key")
	})

	t.Run("Watch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		watchKey := "watch-key"
		watchCh, err := configCenter.Watch(ctx, watchKey, &AppConfig{})
		require.NoError(t, err)
		defer watchCh.Close()

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = configCenter.Set(ctx, watchKey, &AppConfig{Name: "watched-app"})
			time.Sleep(100 * time.Millisecond)
			_ = configCenter.Delete(ctx, watchKey)
		}()

		go func() {
			defer wg.Done()
			// Wait for PUT
			select {
			case event := <-watchCh.Chan():
				assert.Equal(t, config.EventTypePut, event.Type)
			case <-ctx.Done():
				assert.Fail(t, "watcher timed out waiting for PUT event")
				return
			}
			// Wait for DELETE
			select {
			case event := <-watchCh.Chan():
				assert.Equal(t, config.EventTypeDelete, event.Type)
			case <-ctx.Done():
				assert.Fail(t, "watcher timed out waiting for DELETE event")
				return
			}
		}()

		wg.Wait()
	})

	t.Run("WatchPrefix", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		key1 := prefix + "/key1"
		key2 := prefix + "/key2"

		watchCh, err := configCenter.WatchPrefix(ctx, prefix, &AppConfig{})
		require.NoError(t, err)
		defer watchCh.Close()

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			time.Sleep(100 * time.Millisecond)
			_ = configCenter.Set(ctx, key1, &AppConfig{Name: "prefix-app1"})
			time.Sleep(100 * time.Millisecond)
			_ = configCenter.Set(ctx, key2, &AppConfig{Name: "prefix-app2"})
		}()

		go func() {
			defer wg.Done()
			events := 0
			for {
				select {
				case <-watchCh.Chan():
					events++
					if events == 2 {
						return // Success
					}
				case <-ctx.Done():
					assert.Fail(t, "watcher timed out waiting for prefix events")
					return
				}
			}
		}()

		wg.Wait()
		// Use a background context for cleanup to avoid issues with the test context being cancelled
		_ = configCenter.Delete(context.Background(), key1)
		_ = configCenter.Delete(context.Background(), key2)
	})
}
