package appender

import (
	"context"
	"errors"
	"go.uber.org/multierr"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"time"
	"zap_ing/internal/bufferpool"
)

// TODO: check lazy marshalling from fields
// A Field is a marshaling operation used to add a key-value pair to a logger's
// context. Most fields are lazily marshaled, so it's inexpensive to add fields
// to disabled debug-level log statements.

// TODO: message structs could be used in general
type writeMessage struct {
	// TODO: create a custom []byte buffer instance so we do not need to keep the reference to the pool?
	buf   *buffer.Buffer
	ent   zapcore.Entry
	flush chan struct{}
}

type AsyncOption interface {
	apply(*Async) error
}

type asyncOptionsFunc func(*Async) error

func (f asyncOptionsFunc) apply(a *Async) error {
	return f(a)
}

func AsyncMaxQueueLength(length uint32) AsyncOption {
	return asyncOptionsFunc(func(a *Async) error {
		a.maxQueueLength = length
		return nil
	})
}

func AsyncOnQueueNearlyFullForwardTo(fallback Appender) AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		if fallback == nil {
			return errors.New("fallback must not be nil")
		}
		async.fallback = fallback
		return nil
	})
}

func AsyncOnQueueNearlyFullDropMessages() AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		async.fallback = NewDiscard()
		return nil
	})
}

func AsyncQueueMinFreePercent(minFreePercent float32) AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		if minFreePercent < 0 || minFreePercent >= 1 {
			return errors.New("minFreePercent must be between 0 and 1")
		}
		async.calculateDropThresholdFn = func(a *Async) (uint32, error) {
			threshold := float32(async.maxQueueLength) * minFreePercent
			return uint32(threshold), nil
		}
		return nil
	})
}

func AsyncQueueMinFreeItems(minFree uint32) AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		async.calculateDropThresholdFn = func(a *Async) (uint32, error) {
			if a.maxQueueLength < minFree {
				return 0, errors.New("minFree must less than the max queue size")
			}
			return minFree, nil
		}
		return nil
	})
}

func AsyncQueueMonitorPeriod(period time.Duration) AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		if period <= time.Duration(0) {
			return errors.New("period must be positive")
		}
		async.monitorPeriod = period
		return nil
	})
}

func AsyncSyncTimeout(timeout time.Duration) AsyncOption {
	return asyncOptionsFunc(func(async *Async) error {
		if timeout <= time.Duration(0) {
			return errors.New("timeout must be positive")
		}
		async.syncTimeout = timeout
		return nil
	})
}

var _ Appender = &Async{}

type Async struct {
	// only during construction
	maxQueueLength           uint32
	calculateDropThresholdFn func(*Async) (uint32, error)

	// readonly
	primary           Appender
	fallback          Appender
	monitorPeriod     time.Duration
	fallbackThreshold uint32
	syncTimeout       time.Duration

	// state
	queueWrite chan writeMessage
}

func NewAsync(primary Appender, options ...AsyncOption) (a *Async, err error) {
	a = &Async{
		primary: primary,
	}

	AsyncMaxQueueLength(1000).apply(a)
	AsyncQueueMonitorPeriod(time.Second).apply(a)
	AsyncQueueMinFreePercent(.1).apply(a)
	AsyncOnQueueNearlyFullDropMessages().apply(a)

	for _, option := range options {
		err = option.apply(a)
		if err != nil {
			return nil, err
		}
	}

	a.queueWrite = make(chan writeMessage, a.maxQueueLength)
	a.fallbackThreshold, err = a.calculateDropThresholdFn(a)

	a.start()

	return a, err
}

func (a *Async) start() {
	go a.forwardWrite()
	go a.monitorQueueWrite()
}

// the return value n does not work in an async context
func (a *Async) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	msg := writeMessage{
		buf: bufferpool.Get(),
		ent: ent,
	}

	n, err = msg.buf.Write(p)
	if err != nil {
		return
	}

	// this might block shortly until the monitoring routine drops messages
	a.queueWrite <- msg
	return
}

func (m *writeMessage) flushMarker() bool {
	if m.flush == nil {
		return false
	}
	close(m.flush)
	return true
}

func (a *Async) forwardWrite() {
	for {
		select {
		case msg := <-a.queueWrite:
			if msg.flushMarker() {
				continue
			}
			// TODO: handle error
			_, _ = a.primary.Write(msg.buf.Bytes(), msg.ent)
			msg.buf.Free()
		}
	}
}

func (a *Async) monitorQueueWrite() {
	ticker := time.NewTicker(a.monitorPeriod)
	for {
		select {
		case <-ticker.C:
		}
		available := cap(a.queueWrite) - len(a.queueWrite)
		free := a.fallbackThreshold - uint32(available)
		for i := uint32(0); i < free; i++ {
			select {
			case msg := <-a.queueWrite:
				if msg.flushMarker() {
					continue
				}
				// TODO: drop or Fallback: add messageFullStrategy
				a.fallback.Write(msg.buf.Bytes(), msg.ent)
				msg.buf.Free()
			}
		}
	}
}

func (a *Async) Sync() error {
	ctx := context.Background()
	if a.syncTimeout != time.Duration(0) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.syncTimeout)
		defer cancel()
	}
	a.Drain(ctx)
	return multierr.Append(a.primary.Sync(), a.fallback.Sync())
}

// Drain tries to gracefully drain the remaining buffered messages,
// blocking until the buffer is empty or the provided context is cancelled.
func (a *Async) Drain(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	// TODO: also we could use Fallback to drain. add to messageFullStrategy interface
	done := make(chan struct{})
	msg := writeMessage{
		flush: done,
	}
	a.queueWrite <- msg
	select {
	case <-ctx.Done(): // we timed out
	case <-done: // our marker message was handled
	}
}
