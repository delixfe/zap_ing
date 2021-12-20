package tcpwriter

import (
	"errors"
	"math"
	"net"
	"time"
)

var (
	ErrWriteTimeout = errors.New("write timed out")
)

type ConnProviderFn func() (net.Conn, error)
type BackoffFn func(attempt uint64) time.Duration

func DefaultBackoffFn(attempt uint64) time.Duration {
	maxBackoff := time.Second * 30
	base := 2
	factor := time.Second
	backoff := float64(factor) * math.Pow(float64(base), float64(attempt-1))
	// compare float64 values as time.Duration(backoff) produces an overflow for large numbers
	if backoff > float64(maxBackoff) {
		return maxBackoff
	}
	return time.Duration(backoff)
}

// TcpWriter is not thread-safe.
type TcpWriter struct {
	ConnProviderFn ConnProviderFn
	conn           net.Conn
	nowFn          func() time.Time
	writeDeadLine  time.Duration
	// instead of retry, report an error to caller
	WriteTimeout time.Duration
	BackoffFn    BackoffFn
	// retryAttempt we report errors for a given message
	// after WriteTimeout but want to keep the BackoffFn for
	// the next write attempt
	retryAttempt uint64
	// connStale is used by monitor to signal and by getConn to check if a new conn is required
	connStale chan net.Conn
}

// TODO: implement functional options see https://github.com/uber-go/guide/blob/master/style.md#functional-options
func NewTcpWriter(connProviderFn ConnProviderFn) (*TcpWriter, error) {
	// TODO: see https://man7.org/linux/man-pages/man7/tcp.7.html
	// tcp_keepalive_probes -- https://thenotexpert.com/golang-tcp-keepalive/
	// check how to handle with tls connections
	// TODO: config for timeout
	return &TcpWriter{
		ConnProviderFn: connProviderFn,
		writeDeadLine:  time.Second,
		WriteTimeout:   time.Minute * 5,
		BackoffFn:      DefaultBackoffFn,
		nowFn:          time.Now,
		connStale:      make(chan net.Conn),
	}, nil
}

func (w *TcpWriter) Write(p []byte) (n int, err error) {

	// no retry after the deadline
	deadline := w.nowFn().Add(w.WriteTimeout)

	for {

		n, err = w.write(p)

		if err != nil {
			if nerr, ok := err.(net.Error); !ok || nerr.Timeout() || !nerr.Temporary() {
				// permanent error or timeout so close the connection
				w.closeConn()
			}
			// TODO: we block here knowing that the deadline for the current write
			// may already be reached
			w.retrySleep()
			if w.nowFn().After(deadline) {
				// explicitly set written bytes to 0 even if we wrote some bytes
				// as very likely these never reached the target
				return 0, ErrWriteTimeout
			}
			continue
		}

		w.retryReset()

		return
	}
}

func (w *TcpWriter) write(p []byte) (total int, err error) {
	if w.conn == nil {
		w.conn, err = w.getConn()
	}
	if err != nil {
		return
	}

	err = w.conn.SetWriteDeadline(w.nowFn().Add(w.writeDeadLine))
	if err != nil {
		return
	}

	var n int
	for total < len(p) {
		n, err = w.conn.Write(p[total:])
		total += n
		if err != nil {
			return
		}
	}

	return
}

// TODO: consider move retry... in separate type
func (w *TcpWriter) retrySleep() {
	w.retryAttempt += 1
	backoff := w.BackoffFn(w.retryAttempt)
	select {
	case <-time.After(backoff):
	}
}

func (w *TcpWriter) retryReset() {
	w.retryAttempt = 0
}

func (w *TcpWriter) closeConn() {
	if w.conn == nil {
		return
	}
	_ = w.conn.Close()
	w.conn = nil
}

func (w *TcpWriter) Close() (err error) {
	w.closeConn()
	return
}

func (w *TcpWriter) getConn() (net.Conn, error) {
	select {
	case staleConn := <-w.connStale:
		if staleConn == w.conn {
			w.conn = nil
		}
	default:
	}
	if w.conn != nil {
		return w.conn, nil
	}
	conn, err := w.ConnProviderFn()
	if err != nil {
		return nil, err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		keepAlivePeriod := time.Second * 5
		// aggressively set keepalive on the connection
		// this still results in tcp_keepalive_probes * keepAlivePeriod before a broken conn is detected
		// on linux tcp_keepalive_probes = 9
		// see https://man7.org/linux/man-pages/man7/tcp.7.html
		// and see https://groups.google.com/g/golang-nuts/c/IDnJDdM5Ek8 for a discussion on this topic
		// TODO: consider using https://pkg.go.dev/github.com/mikioh/tcp to set tcp_keepalive_probes
		tcpConn.SetKeepAlivePeriod(keepAlivePeriod)
		tcpConn.SetKeepAlive(true)
	}
	go w.monitor(conn)
	return conn, nil
}

// monitor continuously tries to read from the connection to detect socket close.
// This is needed because TCP target uses a write only socket and Linux systems
// take a long time to detect a loss of connectivity on a socket when only writing;
// the writes simply fail without an error returned.
// Copied from https://github.com/mattermost/logr/blob/c356d52ac2edba5368558635e670e8d0fc386672/targets/tcp.go
// TODO: consider using https://groups.google.com/g/golang-nuts/c/IDnJDdM5Ek8 `SIOCOUTQ`
// to query the number of bytes in the socket send queue
func (w *TcpWriter) monitor(conn net.Conn) {
	buf := make([]byte, 1)
	for {
		if conn != w.conn {
			// the monitored conn is not used anymore
			return
		}
		select {
		case <-time.After(1 * time.Second):
		}

		err := conn.SetReadDeadline(w.nowFn().Add(time.Second * 30))
		if err != nil {
			continue
		}

		_, err = conn.Read(buf)

		if errt, ok := err.(net.Error); ok && errt.Timeout() {
			// read timeout is expected, keep looping.
			continue
		}

		// Any other error force a reconnect.
		w.connStale <- conn
		return
	}
}
