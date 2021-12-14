package appender

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type EnvelopingFn func(ent *zapcore.Entry, encoded *buffer.Buffer, output *buffer.Buffer) error

type enveloping struct {
	primary Appender
	envFn   EnvelopingFn
}

func (a *enveloping) Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error) {
	n, err = a.primary.Write(p, ent, fields)
	if err == nil {
		return
	}
	return

}

func (a *enveloping) Append(enc zapcore.Encoder, ent zapcore.Entry, fields []zapcore.Field) (err error) {
	envEnc := &envelopingEncoder{
		inner: enc,
		envFn: a.envFn,
	}
	return a.primary.Append(envEnc, ent, fields)
}

func (a *enveloping) Sync() error {
	return a.Sync()
}
