//go:build !js

package blazewave

import (
	"context"
	"fmt"
	"time"

	"github.com/heyehang/blazewave/core/timer"
)

// initHeartbeat initializes the heartbeat mechanism
func (c *Conn) initHeartbeat() {
	if c.heartbeatConfig.interval > 0 {
		c.lastPongTime.Store(time.Now().Unix())

		// Use provided timer pool or create new one if needed
		if c.heartbeatTimer == nil {
			c.heartbeatTimer = timer.NewTimer(1) // Default capacity for single connection
			c.heartbeatTimerOwned = true         // Mark as locally owned
		}

		c.heartbeatTimerData = c.heartbeatTimer.Add(c.heartbeatConfig.interval, func() {
			if err := c.performHeartbeat(); err != nil {
				c.handleHeartbeatFailure(err)
				return
			}
			// Reschedule the next heartbeat
			if c.heartbeatTimerData != nil {
				c.heartbeatTimer.Set(c.heartbeatTimerData, c.heartbeatConfig.interval)
			}
		})
	}
}

// EnableHeartbeat enables heartbeat with default settings (30s interval, 10s timeout)
func (c *Conn) EnableHeartbeat() {
	c.SetHeartbeat(30*time.Second, 10*time.Second)
}

// DisableHeartbeat disables heartbeat detection
func (c *Conn) DisableHeartbeat() {
	// Disable heartbeat by removing timer
	if c.heartbeatTimerData != nil {
		c.heartbeatTimer.Del(c.heartbeatTimerData)
		c.heartbeatTimerData = nil
	}
}

// SetHeartbeat configures heartbeat detection for the connection
func (c *Conn) SetHeartbeat(interval, timeout time.Duration) {
	if interval <= 0 {
		// Disable heartbeat by removing timer
		if c.heartbeatTimerData != nil {
			c.heartbeatTimer.Del(c.heartbeatTimerData)
			c.heartbeatTimerData = nil
		}
		return
	}

	c.heartbeatConfig.interval = interval
	c.heartbeatConfig.timeout = timeout

	// If connection is already established, restart heartbeat
	if c.heartbeatTimerData != nil {
		c.heartbeatTimer.Del(c.heartbeatTimerData)
		c.heartbeatTimerData = nil
	}

	// Use provided timer pool or create new one if needed
	if c.heartbeatTimer == nil {
		c.heartbeatTimer = timer.NewTimer(1) // Default capacity for single connection
		c.heartbeatTimerOwned = true         // Mark as locally owned
	}

	c.heartbeatTimerData = c.heartbeatTimer.Add(c.heartbeatConfig.interval, func() {
		if err := c.performHeartbeat(); err != nil {
			c.handleHeartbeatFailure(err)
			return
		}
		// Reschedule the next heartbeat
		if c.heartbeatTimerData != nil {
			c.heartbeatTimer.Set(c.heartbeatTimerData, c.heartbeatConfig.interval)
		}
	})
}

// IsHeartbeatEnabled returns whether heartbeat is enabled
func (c *Conn) IsHeartbeatEnabled() bool {
	return c.heartbeatTimerData != nil
}

// GetHeartbeatStatus returns heartbeat status information
func (c *Conn) GetHeartbeatStatus() (enabled bool, interval, timeout time.Duration, lastPong time.Time, failed bool) {
	enabled = c.IsHeartbeatEnabled()
	interval = c.heartbeatConfig.interval
	timeout = c.heartbeatConfig.timeout
	lastPong = time.Unix(c.lastPongTime.Load(), 0)
	failed = c.heartbeatFailed.Load()
	return
}

// IsHeartbeatTimerOwned returns whether the heartbeat timer is owned by this connection
func (c *Conn) IsHeartbeatTimerOwned() bool {
	return c.heartbeatTimerOwned
}

// updateLastPongTime updates the last pong timestamp (called when pong is received)
func (c *Conn) updateLastPongTime() {
	c.lastPongTime.Store(time.Now().Unix())
}

// performHeartbeat sends a ping and waits for pong using the Ping method
func (c *Conn) performHeartbeat() error {
	// Check if we've received a pong recently
	lastPong := c.lastPongTime.Load()
	now := time.Now().Unix()

	if c.heartbeatConfig.timeout > 0 && (now-lastPong) > int64(c.heartbeatConfig.timeout.Seconds()) {
		return WrapConnectionError(fmt.Errorf("heartbeat timeout: no pong received for %v", c.heartbeatConfig.timeout), "heartbeat")
	}

	// Use Ping method with timeout context
	pingCtx, cancel := context.WithTimeout(context.Background(), c.heartbeatConfig.timeout)
	defer cancel()

	// Use the existing Ping method which handles ping/pong logic
	err := c.Ping(pingCtx)
	if err != nil {
		return WrapConnectionError(err, "heartbeat ping failed")
	}

	return nil
}

// handleHeartbeatFailure handles heartbeat failures
func (c *Conn) handleHeartbeatFailure(err error) {
	// Mark heartbeat as failed
	c.heartbeatFailed.Store(true)
	// Close connection due to heartbeat failure
	c.Close(StatusPolicyViolation, "heartbeat timeout")
}
