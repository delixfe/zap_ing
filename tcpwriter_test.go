package zap_ing

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"net"
	"testing"
	"time"
	"zap_ing/test_support"

	"github.com/stretchr/testify/require"
)

func TestTcpWriter_LocalTcpServer_Write(t *testing.T) {

	server, err := test_support.NewLocalTcpServer(100)
	require.NoError(t, err)
	defer server.Close()
	server.Run()

	tcpWriter, err := NewTcpWriter(server.Dial)
	require.NoError(t, err)
	defer assert.NoError(t, tcpWriter.Close())

	message := []byte("message\n")
	count := 50

	for i := 0; i < count; i++ {
		requireWrite(t, tcpWriter, message)
	}
	for i := 0; i < count; i++ {
		requireRead(t, server, message)
	}

	assert.EqualValues(t, 1, server.TotalConnCount(), "expected only one connection to the server")
}

// TODO: flaky test: server.TotalRecLinesCount() < expectedLines
func TestTcpWriter_LocalTcpServer_ReconnectsAfterServerClose(t *testing.T) {
	maxResults := 1000
	server, err := test_support.NewLocalTcpServer(uint(maxResults))
	require.NoError(t, err)
	t.Logf("server at: %v", server.Address())
	defer server.Close()
	server.Run()

	tcpWriter, err := NewTcpWriter(server.Dial)
	require.NoError(t, err)
	defer assert.NoError(t, tcpWriter.Close())

	message := []byte("message\n")

	requireWrite(t, tcpWriter, message)
	requireRead(t, server, message)

	_ = server.CloseAllClientConnections()

	for i := 1; i <= maxResults; i++ {
		requireWrite(t, tcpWriter, []byte(fmt.Sprintf("reconnect %05d\n", i)))
		// TODO: remove sleep when flakiness is improved
		time.Sleep(time.Millisecond * 10)
	}

	//requireRead(t, server, message)
	_, err = server.WaitForOneLineWithTimeout(10)
	assert.NoError(t, err)
	assert.EqualValues(t, 2, server.TotalConnCount(), "TotalConnCount")

	expectedLines := uint32(maxResults - 10)
	if server.TotalRecLinesCount() < expectedLines {
		t.Errorf("expected at least %d lines but got only %d", expectedLines, server.TotalRecLinesCount())
	}

}

func TestTcpWriter_MockConn_AfterWriteTimeout_ReturnErrWriteTimeout(t *testing.T) {
	mockConnection := &test_support.MockConnection{}
	var connProviderFn ConnProviderFn = func() (net.Conn, error) {
		return mockConnection, nil
	}
	writeCalled := 0
	mockConnection.WriteFn = func(b []byte) (int, error) {
		writeCalled++
		return 1, errors.New("some error")
	}
	tcpWriter, err := NewTcpWriter(connProviderFn)
	require.NoError(t, err)
	now := time.Now()
	tcpWriter.WriteTimeout = time.Minute
	tcpWriter.BackoffFn = func(attempt uint64) time.Duration {
		return time.Millisecond
	}
	afterTimeout := now.Add(tcpWriter.WriteTimeout).Add(time.Second)

	tcpWriter.nowFn = test_support.TimeIterator(
		now, now.Add(time.Millisecond*100), now.Add(time.Millisecond*200), afterTimeout)

	message := []byte("message")

	n, err := tcpWriter.Write(message)
	assert.ErrorIs(t, err, ErrWriteTimeout)
	assert.Equal(t, 0, n, "no bytes written")

	assert.Greater(t, writeCalled, 1, "did retry before giving up")

}

func requireWrite(t *testing.T, w io.Writer, data []byte) {
	r := require.New(t)

	n, err := w.Write(data)
	r.NoError(err)
	r.Equal(len(data), n)
}

func requireRead(t *testing.T, server test_support.LocalTcpServer, data []byte) {
	r := require.New(t)

	line, err := server.WaitForOneLineWithTimeout(10)
	if err != nil && errors.Is(err, test_support.ErrWaitTimeout) {
		t.Fatal("WaitForOneLineWithTimeout timed out ")
	}
	r.NoError(err)
	r.EqualValues(data, line)
}

func TestDefaultBackoffFn(t *testing.T) {
	tests := []struct {
		attempt uint64
		want    time.Duration
	}{
		{attempt: 1, want: time.Second},
		{attempt: 2, want: 2 * time.Second},
		{attempt: 3, want: 4 * time.Second},
		{attempt: 4, want: 8 * time.Second},
		{attempt: 5, want: 16 * time.Second},
		{attempt: 6, want: 30 * time.Second},
		{attempt: 1000000, want: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.attempt), func(t *testing.T) {
			if got := DefaultBackoffFn(tt.attempt); got != tt.want {
				t.Errorf("DefaultBackoffFn() = %v, want %v", got, tt.want)
			}
		})
	}
}
