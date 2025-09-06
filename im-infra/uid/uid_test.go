package uid

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				WorkerID:     1,
				DatacenterID: 1,
				EnableUUID:   true,
			},
			wantErr: false,
		},
		{
			name: "workerID too large",
			config: Config{
				WorkerID:     32,
				DatacenterID: 1,
				EnableUUID:   true,
			},
			wantErr: true,
			errMsg:  "workerID must be between 0 and 31",
		},
		{
			name: "negative workerID",
			config: Config{
				WorkerID:     -1,
				DatacenterID: 1,
				EnableUUID:   true,
			},
			wantErr: true,
			errMsg:  "workerID must be between 0 and 31",
		},
		{
			name: "datacenterID too large",
			config: Config{
				WorkerID:     1,
				DatacenterID: 32,
				EnableUUID:   true,
			},
			wantErr: true,
			errMsg:  "datacenterID must be between 0 and 31",
		},
		{
			name: "negative datacenterID",
			config: Config{
				WorkerID:     1,
				DatacenterID: -1,
				EnableUUID:   true,
			},
			wantErr: true,
			errMsg:  "datacenterID must be between 0 and 31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		cfg := DefaultConfig()
		uid, err := New(context.Background(), cfg)
		require.NoError(t, err)
		require.NotNil(t, uid)
		defer uid.Close()
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := Config{
			WorkerID:     32,
			DatacenterID: 1,
			EnableUUID:   true,
		}
		uid, err := New(context.Background(), cfg)
		assert.Error(t, err)
		assert.Nil(t, uid)
	})

	t.Run("with logger option", func(t *testing.T) {
		cfg := DefaultConfig()
		uid, err := New(context.Background(), cfg, WithLogger(nil))
		require.NoError(t, err)
		require.NotNil(t, uid)
		defer uid.Close()
	})

	t.Run("with component name option", func(t *testing.T) {
		cfg := DefaultConfig()
		uid, err := New(context.Background(), cfg, WithComponentName("test-uid"))
		require.NoError(t, err)
		require.NotNil(t, uid)
		defer uid.Close()
	})
}

func TestUID_GenerateInt64(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	t.Run("basic generation", func(t *testing.T) {
		id1 := uid.GenerateInt64()
		id2 := uid.GenerateInt64()

		assert.NotZero(t, id1)
		assert.NotZero(t, id2)
		assert.NotEqual(t, id1, id2)
	})

	t.Run("concurrent generation", func(t *testing.T) {
		const numGoroutines = 100
		const idsPerGoroutine = 100

		idMap := sync.Map{}
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id := uid.GenerateInt64()
					if _, loaded := idMap.LoadOrStore(id, true); loaded {
						t.Errorf("duplicate ID generated: %d", id)
					}
				}
			}()
		}

		wg.Wait()

		count := 0
		idMap.Range(func(key, value interface{}) bool {
			count++
			return true
		})

		assert.Equal(t, numGoroutines*idsPerGoroutine, count, "not all IDs were unique")
	})

	t.Run("monotonic increase", func(t *testing.T) {
		var lastID int64
		for i := 0; i < 100; i++ {
			id := uid.GenerateInt64()
			if lastID != 0 {
				assert.Greater(t, id, lastID, "ID should be monotonically increasing")
			}
			lastID = id
		}
	})
}

func TestUID_GenerateString(t *testing.T) {
	t.Run("UUID mode", func(t *testing.T) {
		cfg := DefaultConfig()
		uid, err := New(context.Background(), cfg)
		require.NoError(t, err)
		defer uid.Close()

		id1 := uid.GenerateString()
		id2 := uid.GenerateString()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)

		assert.Equal(t, 36, len(id1), "UUID should be 36 characters")
		assert.Contains(t, id1, "-", "UUID should contain hyphens")
	})

	t.Run("snowflake string mode", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableUUID = false
		uid, err := New(context.Background(), cfg)
		require.NoError(t, err)
		defer uid.Close()

		id1 := uid.GenerateString()
		id2 := uid.GenerateString()

		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)

		assert.NotContains(t, id1, "-", "Snowflake string should not contain hyphens")
	})
}

func TestUID_GenerateUUIDV4(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	id1 := uid.GenerateUUIDV4()
	id2 := uid.GenerateUUIDV4()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, 36, len(id1), "UUID v4 should be 36 characters")
}

func TestUID_GenerateUUIDV7(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	id1 := uid.GenerateUUIDV7()
	id2 := uid.GenerateUUIDV7()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, 36, len(id1), "UUID v7 should be 36 characters")
}

func TestUID_ValidateUUID(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	invalidUUID := "invalid-uuid-format"

	assert.True(t, uid.ValidateUUID(validUUID))
	assert.False(t, uid.ValidateUUID(invalidUUID))
}

func TestUID_ParseSnowflake(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	id := uid.GenerateInt64()
	timestamp, workerID, datacenterID, sequence := uid.ParseSnowflake(id)

	assert.NotZero(t, timestamp)
	assert.Equal(t, cfg.WorkerID, workerID)
	assert.Equal(t, cfg.DatacenterID, datacenterID)
	assert.GreaterOrEqual(t, sequence, int64(0))
}

func TestUID_Close(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)

	err = uid.Close()
	assert.NoError(t, err)
}

func TestSnowflakeTimestampExtraction(t *testing.T) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(t, err)
	defer uid.Close()

	before := time.Now().UnixMilli()
	id := uid.GenerateInt64()
	after := time.Now().UnixMilli()

	extractedTimestamp := (id >> 22) + 1288834974657

	assert.GreaterOrEqual(t, extractedTimestamp, before)
	assert.LessOrEqual(t, extractedTimestamp, after)
}

func BenchmarkGenerateInt64(b *testing.B) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(b, err)
	defer uid.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = uid.GenerateInt64()
		}
	})
}

func BenchmarkGenerateString(b *testing.B) {
	cfg := DefaultConfig()
	uid, err := New(context.Background(), cfg)
	require.NoError(b, err)
	defer uid.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = uid.GenerateString()
		}
	})
}
