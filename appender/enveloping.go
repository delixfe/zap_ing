package appender

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"zap_ing/internal/bufferpool"
)

// TODO: entry by ref or pointer?
type EnvelopingFn func(p []byte, ent *zapcore.Entry, output *buffer.Buffer) error

type Enveloping struct {
	primary Appender
	envFn   EnvelopingFn
}

func NewEnveloping(inner Appender, envFn EnvelopingFn) *Enveloping {
	return &Enveloping{
		primary: inner,
		envFn:   envFn,
	}
}

func (a *Enveloping) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	buf := bufferpool.Get()
	err = a.envFn(p, &ent, buf)
	if err != nil {
		return
	}
	n, err = a.primary.Write(buf.Bytes(), ent)
	buf.Free()
	return
}

func (a *Enveloping) Sync() error {
	return a.Sync()
}
