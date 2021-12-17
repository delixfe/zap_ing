package appender

import (
	"context"
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
	buf *buffer.Buffer
	ent zapcore.Entry
}

// TODO: we will need a variant with a Fallback appender and one that drops (to use in the Fallback)
type Async struct {
	primary           Appender
	fallback          Appender
	queueWrite        chan writeMessage
	monitorFrequency  time.Duration
	fallbackThreshold int
}

func NewAsync(primary, fallback Appender) *Async {
	async := &Async{
		primary:          primary,
		fallback:         fallback,
		queueWrite:       make(chan writeMessage, 1000),
		monitorFrequency: time.Second,
	}
	async.start()
	return async
}

func (a *Async) start() {
	go a.forwardWrite()
	go a.monitorQueueWrite()
}

// the return value n does not work in an async context
// TODO: we must copy p as we cannot retain it
func (a *Async) Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error) {
	msg := writeMessage{
		buf: bufferpool.Get(),
		ent: ent,
	}
	// TODO: check if buf growths if necessary
	copy(msg.buf.Bytes(), p)

	// this might block shortly until the monitoring routine drops messages
	a.queueWrite <- msg
	return
}

func (a *Async) forwardWrite() {
	for {
		select {
		case msg := <-a.queueWrite:
			// TODO: handle error
			_, _ = a.primary.Write(msg.buf.Bytes(), msg.ent)
			msg.buf.Free()
		}
	}
}

func (a *Async) monitorQueueWrite() {
	for {
		time.Sleep(a.monitorFrequency)
		available := cap(a.queueWrite) - len(a.queueWrite)
		free := a.fallbackThreshold - available
		for i := 0; i < free; i++ {
			select {
			case msg := <-a.queueWrite:
				// TODO: drop or Fallback: add messageFullStrategy
				a.fallback.Write(msg.buf.Bytes(), msg.ent)
				msg.buf.Free()
			}
		}
	}
}

// TODO: here we need some kind of a timeout...
func (a *Async) Sync() error {
	a.Drain(context.TODO())
	return a.Sync()
}

// Drain tries to gracefully drain the remaining buffered messages,
// blocking until the buffer is empty or the provided context is cancelled.
func (a *Async) Drain(ctx context.Context) {
	// TODO: how to know that we have written all in the queue before Drain was called
	// TODO: also we could use Fallback to drain. add to messageFullStrategy interface
}
