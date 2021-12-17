package appender

import (
	"context"
	"go.uber.org/zap/zapcore"
	"time"
)

// TODO: check lazy marshalling from fields
// A Field is a marshaling operation used to add a key-value pair to a logger's
// context. Most fields are lazily marshaled, so it's inexpensive to add fields
// to disabled debug-level log statements.

// TODO: message structs could be used in general
type writeMessage struct {
	p      []byte
	ent    zapcore.Entry
	fields []zapcore.Field
}

type appendMessage struct {
	enc    zapcore.Encoder
	ent    zapcore.Entry
	fields []zapcore.Field
}

// TODO: we will need a variant with a fallback appender and one that drops (to use in the fallback)
type async struct {
	primary           Appender
	fallback          Appender
	queueWrite        chan writeMessage
	queueAppend       chan appendMessage
	monitorFrequency  time.Duration
	fallbackThreshold int
}

// the return value n does not work in an async context
// TODO: we must copy p as we cannot retain it
func (a *async) Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error) {
	// this might block shortly until the monitoring routine drops messages
	a.queueWrite <- writeMessage{
		// TODO: copy p
		p:      p,
		ent:    ent,
		fields: fields}
	return
}

// TODO: fields might hold references to mutable values like pooled []byte
//	analyze it that could happen and how we would mitigate
// 	- can happen
//  - will happen
//  - mitigation: render already as json :(
func (a *async) Append(enc zapcore.Encoder, ent zapcore.Entry, fields []zapcore.Field) (err error) {
	// this might block shortly until the monitoring routine drops messages
	a.queueAppend <- appendMessage{
		enc:    enc,
		ent:    ent,
		fields: fields}
	return
}

func (a *async) forwardWrite() {
	for {
		select {
		case msg := <-a.queueWrite:
			_, _ = a.primary.Write(msg.p, msg.ent, msg.fields)
		}
	}
}

func (a *async) monitorQueueWrite() {
	for {
		time.Sleep(a.monitorFrequency)
		available := cap(a.queueWrite) - len(a.queueWrite)
		free := a.fallbackThreshold - available
		for i := 0; i < free; i++ {
			select {
			case msg := <-a.queueWrite:
				// TODO: drop or fallback
				a.fallback.Write(msg.p, msg.ent, msg.fields)
			}
		}
	}
}

// TODO: here we need some kind of a timeout...
func (a *async) Sync() error {
	a.Drain(context.TODO())
	return a.Sync()
}

// Drain tries to gracefully drain the remaining buffered messages,
// blocking until the buffer is empty or the provided context is cancelled.
func (a *async) Drain(ctx context.Context) {

}
