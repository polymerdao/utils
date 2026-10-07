//go:build linux

// Package netx holds network helpers that the standard net package doesn't have.
package netx

import (
	"crypto/tls"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// SetTCPUserTimeout makes the kernel drop conn when sent data stays unacknowledged for timeout, instead of
// waiting for its ~15 min retry limit (e.g. when the peer's node is gone). Non-TCP connections are left alone.
func SetTCPUserTimeout(conn net.Conn, timeout time.Duration) error {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		conn = tlsConn.NetConn()
	}
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil
	}
	raw, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}
	var sockErr error
	ctrlfn := func(fd uintptr) {
		sockErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, int(timeout.Milliseconds()))
	}
	if err := raw.Control(ctrlfn); err != nil {
		return err
	}
	return sockErr
}
