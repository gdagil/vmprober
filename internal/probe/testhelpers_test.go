package probe

import (
	"net"
	"testing"

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

// closedUDPAddr reserves an ephemeral UDP port on the loopback interface and
// releases it, returning an address with no listener behind it.
func closedUDPAddr(t *testing.T) *net.UDPAddr {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := pc.LocalAddr().(*net.UDPAddr)
	require.NoError(t, pc.Close())
	return addr
}
