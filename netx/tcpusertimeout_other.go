//go:build !linux

// Package netx holds network helpers that the standard net package doesn't have.
package netx

import (
	"net"
	"time"
)

// SetTCPUserTimeout does nothing: TCP_USER_TIMEOUT is Linux only.
func SetTCPUserTimeout(net.Conn, time.Duration) error {
	return nil
}
