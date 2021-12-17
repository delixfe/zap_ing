package chaos

import (
	"context"
	"go.uber.org/zap/zapcore"
	"sync/atomic"
	"testing"
	"time"
	"zap_ing/appender"
)

func TestBlockingSwitchable_Break(t *testing.T) {
	written := uint32(0)
	ctx := context.Background()
	if deadline, ok := t.Deadline(); ok {
		ctx, _ = context.WithDeadline(context.Background(), deadline)
	}

	inner := appender.NewDelegating(func(p []byte, ent zapcore.Entry) (n int, err error) {
		atomic.AddUint32(&written, 1)
		return len(p), nil
	}, nil)

	blocking := NewBlockingSwitchable(ctx, inner)
	blocking.Write([]byte{}, zapcore.Entry{})
	if written != 1 {
		t.Fatal("expected 1 write")
	}

	blocking.Break()

	go func() {
		blocking.Write([]byte{}, zapcore.Entry{})
	}()

	time.Sleep(time.Millisecond * 100)
	if written != 1 {
		t.Fatal("expected no further write while blocking")
	}

	blocking.Fix()

	time.Sleep(time.Millisecond * 100)

	if written != 2 {
		t.Errorf("expected 2nd write after unblocking, written is %d", written)
	}
}
