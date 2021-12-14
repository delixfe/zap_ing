package appender

import (
	"go.uber.org/zap/zapcore"
)

var _ Appender = &writer{}

type writer struct {
	out zapcore.WriteSyncer
}

func (a *writer) Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error) {
	return a.out.Write(p)
}

func (a *writer) Append(enc zapcore.Encoder, ent zapcore.Entry, fields []zapcore.Field) (err error) {
	buf, err := enc.EncodeEntry(ent, fields)
	if err != nil {
		return
	}
	defer buf.Free()
	_, err = a.out.Write(buf.Bytes())
	return err
}

func (a *writer) Sync() error {
	return a.Sync()
}
