package test_support

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestLocalTcpServer(t *testing.T) {
	t.Parallel()
	t.Run("no_connection", func(t *testing.T) {
		server, err := NewLocalTcpServer(100)
		server.Run()
		require.NoError(t, err)
		t.Logf("address: %v", server.Address())
		server.Close()
		// should not run in timeout
		assert.EqualValues(t, 0, server.TotalConnCount(), "totalConnCount")
	})
	t.Run("connection", func(t *testing.T) {
		server, err := NewLocalTcpServer(100)
		require.NoError(t, err)
		server.Run()
		t.Logf("address: %v", server.Address())
		conn, err := server.Dial()
		require.NoError(t, err)
		defer func(conn net.Conn) {
			_ = conn.Close()
		}(conn)
		_, err = conn.Write([]byte("message\n"))
		require.NoError(t, err)
		_, err = server.WaitForOneLine()
		assert.NoError(t, err)
		server.Close()
		// should not run in timeout
		assert.EqualValues(t, 1, server.TotalConnCount(), "TotalConnCount")
		assert.EqualValues(t, uint64(1), server.TotalRecLinesCount(), "TotalRecLinesCount")
	})
}
