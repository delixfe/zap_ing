package appender

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

var _ Appender = &fallback{}

type fallback struct {
	primary  Appender
	fallback Appender
}

func (a *fallback) Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error) {

	n, primErr := a.primary.Write(p, ent, fields)
	if primErr == nil {
		return n, nil
	}
	n, fallErr := a.fallback.Write(p, ent, fields)
	if fallErr == nil {
		return n, nil
	}

	// TODO: decide which error to return
	return n, multierr.Append(primErr, fallErr)

}

func (a *fallback) Append(enc zapcore.Encoder, ent zapcore.Entry, fields []zapcore.Field) error {
	primErr := a.primary.Append(enc, ent, fields)
	if primErr == nil {
		return nil
	}
	fallErr := a.fallback.Append(enc, ent, fields)
	if fallErr == nil {
		return nil
	}

	// TODO: decide which error to return
	return multierr.Append(primErr, fallErr)
}

func (a *fallback) Sync() error {
	return multierr.Append(a.primary.Sync(), a.fallback.Sync())
}
