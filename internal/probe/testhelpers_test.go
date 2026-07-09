package probe

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// closedTCPAddr reserves an ephemeral TCP port on the loopback interface and
// immediately releases it, returning an address that is (almost certainly) not
// accepting connections. This enables deterministic connection-failure tests
// without depending on external network reachability.
func closedTCPAddr(t *testing.T) *net.TCPAddr {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().(*net.TCPAddr)
	require.NoError(t, ln.Close())
	return addr
}

// startUDPServer starts a loopback UDP server on 127.0.0.1:0 and returns its
// address plus a cleanup function. Fully hermetic.
//
// When echo is true it mirrors every datagram back to its sender after
// replyDelay, which lets callers assert a deterministic lower-bound RTT. When
// echo is false it reads and discards datagrams and never replies, so a client
// deterministically hits its read/response timeout (as opposed to receiving an
// ICMP "port unreachable" from a closed port). replyDelay is ignored when echo
// is false.
func startUDPServer(t *testing.T, echo bool, replyDelay time.Duration) (*net.UDPAddr, func()) {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	require.NoError(t, err)

	go func() {
		buffer := make([]byte, 4096)
		for {
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return
			}
			if !echo {
				continue // silent: intentionally never reply
			}
			if replyDelay > 0 {
				time.Sleep(replyDelay)
			}
			_, _ = conn.WriteToUDP(buffer[:n], clientAddr)
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr), func() {
		_ = conn.Close()
	}
}
