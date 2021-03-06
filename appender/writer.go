package appender

import (
	"go.uber.org/zap/zapcore"
	"syscall"
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
	// ignore non-actionable errors
	// as per https://github.com/open-telemetry/opentelemetry-collector/issues/4153
	// and https://github.com/open-telemetry/opentelemetry-collector/blob/2116719e97eb232a692364b51454620712823a89/exporter/loggingexporter/known_sync_error.go#L35
	// TODO: windows implementation
	err := a.out.Sync()
	switch err {
	case syscall.EINVAL, syscall.ENOTSUP, syscall.ENOTTY, syscall.EBADF:
		return nil
	}
	return err
}

func (a *Writer) Synchronized() bool {
	return true
}
