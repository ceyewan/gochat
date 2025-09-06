package internal

import (
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/google/uuid"
)

const (
	workerIDBits     = 5
	datacenterIDBits = 5
	sequenceBits     = 12

	maxWorkerID     = -1 ^ (-1 << workerIDBits)
	maxDatacenterID = -1 ^ (-1 << datacenterIDBits)
	maxSequence     = -1 ^ (-1 << sequenceBits)

	workerIDShift     = sequenceBits
	datacenterIDShift = sequenceBits + workerIDBits
	timestampShift    = sequenceBits + workerIDBits + datacenterIDBits

	// Twitter epoch (November 4, 2010, 01:42:54 UTC)
	twepoch = int64(1288834974657)
)

type Client struct {
	mu            sync.Mutex
	logger        clog.Logger
	lastTimestamp int64
	workerID      int64
	datacenterID  int64
	sequence      int64
	enableUUID    bool
}

func NewClient(cfg interface {
	GetWorkerID() int64
	GetDatacenterID() int64
	GetEnableUUID() bool
}, logger clog.Logger) (*Client, error) {
	workerID := cfg.GetWorkerID()
	datacenterID := cfg.GetDatacenterID()

	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("worker ID must be between 0 and %d", maxWorkerID)
	}
	if datacenterID < 0 || datacenterID > maxDatacenterID {
		return nil, fmt.Errorf("datacenter ID must be between 0 and %d", maxDatacenterID)
	}

	return &Client{
		logger:       logger,
		workerID:     workerID,
		datacenterID: datacenterID,
		enableUUID:   cfg.GetEnableUUID(),
	}, nil
}

func (c *Client) GenerateInt64() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	timestamp := c.currentTimestamp() - twepoch

	if timestamp < c.lastTimestamp {
		// Instead of sleeping, wait for the next millisecond
		timestamp = c.waitNextMillis(c.lastTimestamp)
		timestamp = timestamp - twepoch
	}

	if c.lastTimestamp == timestamp {
		c.sequence = (c.sequence + 1) & maxSequence
		if c.sequence == 0 {
			timestamp = c.waitNextMillis(timestamp)
			timestamp = timestamp - twepoch
		}
	} else {
		c.sequence = 0
	}

	c.lastTimestamp = timestamp

	return ((timestamp) << timestampShift) |
		((c.datacenterID) << datacenterIDShift) |
		((c.workerID) << workerIDShift) |
		c.sequence
}

func (c *Client) GenerateString() string {
	if c.enableUUID {
		return c.generateUUID()
	}

	return fmt.Sprintf("%d", c.GenerateInt64())
}

func (c *Client) generateUUID() string {
	id := uuid.New()
	return id.String()
}

func (c *Client) GenerateUUIDV4() string {
	return uuid.NewString()
}

func (c *Client) GenerateUUIDV7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to UUID v4 if V7 fails
		return uuid.NewString()
	}
	return id.String()
}

func (c *Client) ValidateUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}

func (c *Client) ParseSnowflake(id int64) (timestamp int64, workerID int64, datacenterID int64, sequence int64) {
	timestamp = (id >> timestampShift) + twepoch
	workerID = (id >> workerIDShift) & maxWorkerID
	datacenterID = (id >> datacenterIDShift) & maxDatacenterID
	sequence = id & maxSequence
	return
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) currentTimestamp() int64 {
	return time.Now().UnixMilli()
}

func (c *Client) waitNextMillis(currentTimestamp int64) int64 {
	timestamp := c.currentTimestamp()
	for timestamp <= currentTimestamp {
		timestamp = c.currentTimestamp()
	}
	return timestamp
}

type Config interface {
	GetWorkerID() int64
	GetDatacenterID() int64
	GetEnableUUID() bool
}
