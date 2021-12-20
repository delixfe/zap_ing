package test_support

import (
	"net"
	"time"
)

var _ net.Conn = &MockConnection{}

type MockConnection struct {
	ReadFn  func(b []byte) (int, error)
	WriteFn func(b []byte) (int, error)
}

func (c *MockConnection) Read(b []byte) (int, error) {
	readFn := c.ReadFn
	if readFn != nil {
		return readFn(b)
	}
	return len(b), nil
}

func (c *MockConnection) Write(b []byte) (int, error) {
	writeFn := c.WriteFn
	if writeFn != nil {
		return writeFn(b)
	}
	return len(b), nil
}
func (c *MockConnection) Close() error { return nil }

func (c *MockConnection) LocalAddr() net.Addr { return &mockAddr{} }

func (c *MockConnection) RemoteAddr() net.Addr { return &mockAddr{} }

func (c *MockConnection) SetDeadline(_ time.Time) error { return nil }

func (c *MockConnection) SetReadDeadline(_ time.Time) error { return nil }

func (c *MockConnection) SetWriteDeadline(_ time.Time) error { return nil }

func (c *MockConnection) SetReadBuffer(_ int) error { return nil }

func (c *MockConnection) SetWriteBuffer(_ int) error { return nil }

type Addr interface {
	Network() string // name of the network (for example, "tcp", "udp")
	String() string  // string form of address (for example, "192.0.2.1:25", "[2001:db8::1]:80")
}

type mockAddr struct {
	network    string
	addrstring string
}

func (m *mockAddr) Network() string {
	return m.network
}
func (m *mockAddr) String() string {
	return m.addrstring
}
