package appender

import (
	"context"
	"fmt"
	"go.uber.org/zap/zapcore"
	"io"
	"sync/atomic"
	"testing"
	"time"
	"zap_ing/appender/appendercore"
	"zap_ing/appender/chaos"
)

func NewTestFailOnWriteAppender(t *testing.T) appendercore.Appender {
	return NewDelegating(func(_ []byte, _ zapcore.Entry) (n int, err error) {
		t.Fatal("write called on TestFailOnWriteAppender")
		return 0, nil
	}, nil, true)
}

func NewWriteCountingAppender() (appendercore.Appender, func() uint64) {
	counter := uint64(0)
	writeFn := func(p []byte, _ zapcore.Entry) (n int, err error) {
		atomic.AddUint64(&counter, uint64(1))
		return len(p), nil
	}
	loadCounterFn := func() uint64 {
		return atomic.LoadUint64(&counter)
	}
	return NewDelegating(writeFn, nil, true), loadCounterFn
}

func Write(a appendercore.Appender) error {
	_, err := a.Write([]byte{}, zapcore.Entry{})
	return err
}

func AssertWrittenEquals(t *testing.T, expected uint64, written func() uint64, msg string) {
	actual := written()
	if actual != expected {
		t.Helper()
		t.Errorf("%s: \n\texpected writes: %d\n\tactual   writes: %d", msg, expected, actual)
	}
}

func TestAsync_smoke(t *testing.T) {
	ctx := context.Background()
	counting, written := NewWriteCountingAppender()
	blocking := chaos.NewBlockingSwitchableCtx(ctx, counting)

	async, _ := NewAsync(blocking, AsyncOnQueueNearlyFullForwardTo(NewTestFailOnWriteAppender(t)))
	Write(async)
	async.Sync()
	AssertWrittenEquals(t, 1, written, "after sync")

	blocking.Break()
	Write(async)

	AssertWrittenEquals(t, 1, written, "breaking, message should be enqueued")

	blocking.Fix()
	async.Sync()
	AssertWrittenEquals(t, 2, written, "fixed, message be forwarded")
}

type expectCounters struct {
	primary  uint64
	fallback uint64
	blocked  uint64
	errors   uint64
}

func (e expectCounters) String() string {
	return fmt.Sprintf(
		"expectedCounters: \n\tprimary:  %d\n\tfallback: %d\n\tbroken:  %d\n\terrors:   %d",
		e.primary, e.fallback, e.blocked, e.errors)
}

type actualAccessors struct {
	primary  func() uint64
	fallback func() uint64
	blocked  func() uint64
	errors   func() uint64
}

func (e actualAccessors) String() string {
	return fmt.Sprintf(
		"expectedCounters: \n\tprimary:  %d\n\tfallback: %d\n\tbroken:  %d\n\terrors:   %d",
		e.primary(), e.fallback(), e.blocked(), e.errors())
}

func AssertCounters(t *testing.T, expect expectCounters, accessors actualAccessors, msg string) {
	t.Helper()
	AssertWrittenEquals(t, expect.primary, accessors.primary, msg+" primary")
	AssertWrittenEquals(t, expect.fallback, accessors.fallback, msg+" fallback")
	AssertWrittenEquals(t, expect.blocked, accessors.blocked, msg+" broken")
	AssertWrittenEquals(t, expect.errors, accessors.errors, msg+" errors")
}

func TestAsync(t *testing.T) {

	type args struct {
		queueLength int
		threshold   int
		write       int
		options     []AsyncOption
		broken      expectCounters
		fixed       expectCounters
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "mini", args: args{
			queueLength: 1,
			threshold:   1,
			write:       2,
			broken:      expectCounters{primary: 0, fallback: 1}, // one is consumed by blocking
			fixed:       expectCounters{primary: 1, fallback: 1},
		}},
		{name: "example", args: args{
			queueLength: 10,
			threshold:   2,
			write:       10,
			broken:      expectCounters{primary: 0, fallback: 1}, // one is consumed by blocking
			fixed:       expectCounters{primary: 9, fallback: 1},
		}},
		{name: "writes equal length", args: args{
			queueLength: 100,
			threshold:   10,
			write:       100,
			broken:      expectCounters{primary: 0, fallback: 9}, // one is consumed by blocking
			fixed:       expectCounters{primary: 91, fallback: 9},
		}},
		{name: "no monitor, more writes than length -> block", args: args{
			queueLength: 1,
			threshold:   0,
			write:       10,
			options:     []AsyncOption{AsyncQueueMonitorPeriod(time.Hour)},
			broken:      expectCounters{primary: 0, fallback: 0, blocked: 1}, // one is consumed by blocking, 1 in queue
			fixed:       expectCounters{primary: 10, fallback: 0},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			primary, primaryCounter := NewWriteCountingAppender()
			blocking := chaos.NewBlockingSwitchable(primary)
			blocking.Break()

			fallback, fallbackCounter := NewWriteCountingAppender()
			options := []AsyncOption{
				AsyncOnQueueNearlyFullForwardTo(fallback),
				AsyncMaxQueueLength(tt.args.queueLength),
				AsyncQueueMinFreeItems(tt.args.threshold),
				AsyncQueueMonitorPeriod(time.Millisecond),
			}
			async, _ := NewAsync(blocking,
				append(options, tt.args.options...)...,
			)
			defer async.Shutdown(context.Background())
			blocked := uint64(0)
			errors := uint64(0)
			actual := actualAccessors{
				primary:  primaryCounter,
				fallback: fallbackCounter,
				blocked:  func() uint64 { return atomic.LoadUint64(&blocked) },
				errors:   func() uint64 { return atomic.LoadUint64(&errors) },
			}
			go func() {
				for i := 0; i < tt.args.write; i++ {
					atomic.AddUint64(&blocked, 1)
					if Write(async) != nil {
						atomic.AddUint64(&errors, 1)
					}
					atomic.AddUint64(&blocked, ^uint64(0))
				}
			}()

			time.Sleep(time.Millisecond * 10) // give monitor time to catch up
			AssertCounters(t, tt.args.broken, actual, "broken")

			blocking.Fix()
			time.Sleep(time.Millisecond * 10) // give monitor time to catch up
			async.Drain(context.Background())
			AssertCounters(t, tt.args.fixed, actual, "fixed")

		})
	}
}

func TestAsync_Write_afterShutdown_returnsErr(t *testing.T) {
	primary, primaryCounter := NewWriteCountingAppender()
	fallback, fallbackCounter := NewWriteCountingAppender()

	async, _ := NewAsync(primary,
		AsyncOnQueueNearlyFullForwardTo(fallback),
	)
	async.Shutdown(context.Background())

	err := Write(async)

	if err == nil {
		t.Error("expected an error")
	}
	AssertWrittenEquals(t, 0, primaryCounter, "primary")
	AssertWrittenEquals(t, 0, fallbackCounter, "fallback")
}

type assertFn func(*Async) bool

func TestNewAsync(t *testing.T) {

	tests := []struct {
		name       string
		options    []AsyncOption
		wantErr    bool
		assertions []assertFn
	}{
		{name: "forwardTo nil fallback", wantErr: true,
			options: []AsyncOption{AsyncOnQueueNearlyFullForwardTo(nil)}},
		{name: "forwardTo fallback is synchronized",
			options:    []AsyncOption{AsyncOnQueueNearlyFullForwardTo(NewWriter(zapcore.AddSync(io.Discard)))},
			assertions: []assertFn{func(a *Async) bool { return appendercore.Synchronized(a) }},
		},
		{name: "max queue length negative", wantErr: true, options: []AsyncOption{AsyncMaxQueueLength(-1)}},
		{name: "queue monitor period negative", wantErr: true, options: []AsyncOption{AsyncQueueMonitorPeriod(-1 * time.Second)}},
		{name: "queue monitor period zero", wantErr: true, options: []AsyncOption{AsyncQueueMonitorPeriod(0)}},
		{name: "async sync timeout negative", wantErr: true, options: []AsyncOption{AsyncSyncTimeout(-1 * time.Second)}},
		{name: "sync sync timeout zero", wantErr: true, options: []AsyncOption{AsyncSyncTimeout(0)}},
		{name: "min free items lt zero", wantErr: true, options: []AsyncOption{
			AsyncQueueMinFreeItems(-1),
		}},
		{name: "min free items greater queue length", wantErr: true, options: []AsyncOption{
			AsyncQueueMinFreeItems(100),
			AsyncMaxQueueLength(10),
		}},
		{name: "min free percent lt 0", wantErr: true, options: []AsyncOption{
			AsyncQueueMinFreePercent(-1)}},
		{name: "min free percent gt 1", wantErr: true, options: []AsyncOption{
			AsyncQueueMinFreePercent(-1)}},

		{name: "min free percent calculation 10",
			options: []AsyncOption{
				AsyncQueueMinFreePercent(0.1),
				AsyncMaxQueueLength(100),
			},
			assertions: []assertFn{func(a *Async) bool { return a.fallbackThreshold == 10 }},
		},
		{name: "min free percent calculation 20",
			options: []AsyncOption{
				AsyncQueueMinFreePercent(0.2),
				AsyncMaxQueueLength(10),
			},
			assertions: []assertFn{func(a *Async) bool { return a.fallbackThreshold == 2 }},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primary := NewDiscard()
			gotA, err := NewAsync(primary, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAsync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, assertion := range tt.assertions {
				if !assertion(gotA) {
					t.Error("assertion failed")
				}
			}
		})
	}
}

func TestNewAsync_nilPrimary_returnsErr(t *testing.T) {
	async, err := NewAsync(nil)
	if err == nil {
		t.Error("expected an error")
	}
	if async != nil {
		t.Errorf("did not expcect an appender")
	}
}
