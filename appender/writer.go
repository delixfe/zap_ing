package appender

import (
	"go.uber.org/zap/zapcore"
)

var _ Appender = &Writer{}

type Writer struct {
	out zapcore.WriteSyncer
}

func NewWriter(out zapcore.WriteSyncer) *Writer {
	return &Writer{out: out}
}

func (a *Writer) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	return a.out.Write(p)
}

func (a *Writer) Sync() error {
	return a.Sync()
}
