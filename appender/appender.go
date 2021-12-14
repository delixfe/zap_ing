package appender

import (
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = &appenderCore{}

// Appender
// TODO: decide on variant, especially in regard of pooling
// 1. Write with p, ent, fields
// 2. Write with p, ent
// 3. Write with p and a subset of ent
// A. Append with enc, ent, fields
type Appender interface {

	// Write
	// must not retain p
	Write(p []byte, ent zapcore.Entry, fields []zapcore.Field) (n int, err error)

	Append(enc zapcore.Encoder, ent zapcore.Entry, fields []zapcore.Field) (err error)

	// Sync flushes buffered logs (if any).
	Sync() error
}

type appenderCore struct {
	zapcore.LevelEnabler
	enc      zapcore.Encoder
	appender Appender
}

func (c *appenderCore) With(fields []zapcore.Field) zapcore.Core {
	enc := c.enc.Clone()
	for i := range fields {
		fields[i].AddTo(enc)
	}
	return &appenderCore{
		LevelEnabler: c.LevelEnabler,
		appender:     c.appender,
		enc:          enc,
	}
}

func (c *appenderCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}
func (c *appenderCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	_, err = c.appender.Write(buf.Bytes(), ent, fields)
	buf.Free()
	if err != nil {
		return err
	}
	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		_ = c.Sync()
	}
	return nil
}

func (c *appenderCore) Sync() error {
	return c.appender.Sync()
}
