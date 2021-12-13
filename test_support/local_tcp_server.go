package test_support

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func NewLocalListener(network string) (net.Listener, error) {
	// check https://cs.opensource.google/go/x/net/+/012df41e:nettest/nettest.go
	switch network {
	case "tcp":
		if ln, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
			return ln, nil
		}

		if ln, err := net.Listen("tcp6", "[::1]:0"); err == nil {
			return ln, nil
		} else {
			return nil, fmt.Errorf("listener could not be created: %w", err)
		}
	}

	return nil, fmt.Errorf("network %s is currently not support", network)
}

var (
	ErrServerClosed = errors.New("server is closed")
	ErrWaitTimeout  = errors.New("wait timed out")
)

type LocalTcpServer interface {
	Address() string
	Dial() (net.Conn, error)
	Close()
	Run()
	WaitForOneLine() ([]byte, error)
	WaitForOneLineWithTimeout(seconds int) ([]byte, error)
	CloseAllClientConnections() error
	TotalConnCount() int
	TotalRecLinesCount() uint32
}

func NewLocalTcpServer(maxResultsQueue uint) (LocalTcpServer, error) {
	listener, err := NewLocalListener("tcp")
	if err != nil {
		return nil, err
	}
	return &localTcpServer{
		listener:    listener,
		closedChan:  make(chan struct{}, 1),
		resultsChan: make(chan TcpServerResult, maxResultsQueue),
		activeConn:  make(map[net.Conn]struct{}),
	}, nil
}

type localTcpServer struct {
	listener           net.Listener
	closedChan         chan struct{}
	resultsChan        chan TcpServerResult
	activeConn         map[net.Conn]struct{}
	totalConnCount     int
	totalRecLinesCount uint32
	mu                 sync.Mutex
}

type TcpServerResult struct {
	Line  []byte
	Conn  *net.Conn
	Error error
}

func (s *localTcpServer) Address() string {
	return s.listener.Addr().String()
}

func (s *localTcpServer) TotalConnCount() int {
	return s.totalConnCount
}

func (s *localTcpServer) TotalRecLinesCount() uint32 {
	return s.totalRecLinesCount
}

func (s *localTcpServer) Dial() (net.Conn, error) {
	return net.Dial("tcp", s.Address())
}

func (s *localTcpServer) Close() {
	close(s.closedChan)
	_ = s.listener.Close()
}

func (s *localTcpServer) Run() {
	go s.handleConnections()
}

func (s *localTcpServer) handleConnections() {
	for {
		select {
		case <-s.closedChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.closedChan:
					return
				default:
					log.Fatalf("could not accept connection: %v", err)
				}
			}
			go s.handleConnection(conn)
		}

	}
}

func (s *localTcpServer) handleConnection(conn net.Conn) {
	s.trackConn(conn, true)
	defer closeConnIgnoringErrors(conn)
	defer s.trackConn(conn, false)

	reader := bufio.NewReader(conn)

	for {
		select {
		case <-s.closedChan:
			return
		default:
			// TODO: add timeout
			//conn.SetDeadline(time.Now().Add(time.Second))

			line, err := reader.ReadBytes('\n')
			if errors.Is(err, net.ErrClosed) {
				return
			}

			if err == nil {
				atomic.AddUint32(&s.totalRecLinesCount, 1)
			}

			s.resultsChan <- TcpServerResult{
				Line:  line,
				Conn:  &conn,
				Error: err,
			}

		}
	}
}

func (s *localTcpServer) trackConn(c net.Conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	//log.Printf("trackConn %v: %v", add, c.RemoteAddr().String())
	if s.activeConn == nil {
		s.activeConn = make(map[net.Conn]struct{})
	}
	if add {
		s.totalConnCount += 1
		s.activeConn[c] = struct{}{}
	} else {
		delete(s.activeConn, c)
	}
}

// WaitForOneLine waits for a line to be received.
func (s *localTcpServer) WaitForOneLine() ([]byte, error) {
	select {
	case <-s.closedChan:
		return nil, ErrServerClosed
	case result := <-s.resultsChan:
		return result.Line, result.Error
	}
}

// WaitForOneLineWithTimeout waits for a line to be received with a timeout.
func (s *localTcpServer) WaitForOneLineWithTimeout(seconds int) ([]byte, error) {
	select {
	case <-s.closedChan:
		return nil, ErrServerClosed
	case result := <-s.resultsChan:
		return result.Line, result.Error
	case <-time.After(time.Duration(seconds) * time.Second):
		return nil, ErrWaitTimeout
	}
}

func (s *localTcpServer) CloseAllClientConnections() error {
	// TODO: several variants
	// see https://itnext.io/forcefully-close-tcp-connections-in-golang-e5f5b1b14ce6
	// TODO: how to simulate network interrupt?
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.activeConn {
		closeConnIgnoringErrors(c)
	}
	return nil
}

func closeConnIgnoringErrors(conn net.Conn) {
	if conn == nil {
		return
	}
	_ = conn.Close()
}
