package core

import (
	"sync/atomic"
	"time"
)

/*
Metrics Tracker

In a high-performance single-threaded server, keeping track of metrics without 
slowing down the hot path is critical. We use atomic operations for counters
so that the background thread (like AOF rewrites) or metrics-gathering routines
can read them safely without locks, even though command execution itself is single-threaded.
*/

type ServerMetrics struct {
	StartTime                time.Time
	CommandsProcessed        uint64
	ConnectionsReceived      uint64
	NetworkBytesIn           uint64
	NetworkBytesOut          uint64
}

var GlobalMetrics ServerMetrics

func init() {
	GlobalMetrics.StartTime = time.Now()
}

// TrackCommand increments the command counter.
// Should be called for every command executed.
func TrackCommand() {
	atomic.AddUint64(&GlobalMetrics.CommandsProcessed, 1)
}

// TrackConnection increments the connection counter.
func TrackConnection() {
	atomic.AddUint64(&GlobalMetrics.ConnectionsReceived, 1)
}

// GetUptime returns the server uptime in seconds.
func GetUptime() int64 {
	return int64(time.Since(GlobalMetrics.StartTime).Seconds())
}

// OpsPerSecond calculates the average ops/sec over the server's lifetime.
// In a real implementation, this would track instantaneous ops/sec using
// a ring buffer or exponential moving average updated via the cron loop.
func OpsPerSecond() float64 {
	uptime := time.Since(GlobalMetrics.StartTime).Seconds()
	if uptime == 0 {
		return 0
	}
	return float64(atomic.LoadUint64(&GlobalMetrics.CommandsProcessed)) / uptime
}
