package appender

import (
	"go.uber.org/zap/zapcore"
	"zap_ing/appender/appendercore"
)

var _ appendercore.Appender = &Writer{}

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

func (a *Writer) Synchronized() bool {
	return true
}
