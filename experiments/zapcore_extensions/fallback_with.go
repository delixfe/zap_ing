package zapcore_extensions

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = &fallbackOptimizedForWith{}

// TODO: nameing of CloneWithEncoderCore
type CloneWithEncoderCore interface {
	zapcore.Core
	CloneWithEncoder(encoder zapcore.Encoder) zapcore.Core
}

// With has no extra cost
// Write has at least one extra allocation for primary clone
// fields on primary and fallback set with `With(fields []zapcore.Field)` will be ignored
// sub Cores must confirm to CloneWithEncoderCore
type fallbackOptimizedForWith struct {
	zapcore.LevelEnabler
	enc zapcore.Encoder
	// fields set with `With(fields []zapcore.Field)` will be ignored
	// check will not be called
	// level will be ignored
	primary CloneWithEncoderCore
	// fields set with `With(fields []zapcore.Field)` will be ignored
	fallback CloneWithEncoderCore
}

func (c *fallbackOptimizedForWith) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	addFields(clone.enc, fields)
	return clone
}

func (c *fallbackOptimizedForWith) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}
func (c *fallbackOptimizedForWith) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	// this means at least one allocation per Write
	primErr := c.primary.CloneWithEncoder(c.enc).Write(ent, fields)
	if primErr == nil {
		return nil
	}
	fallErr := c.fallback.CloneWithEncoder(c.enc).Write(ent, fields)
	if fallErr == nil {
		return nil
	}

	// TODO: decide which error to return
	return multierr.Append(primErr, fallErr)
}

func (c *fallbackOptimizedForWith) Sync() error {
	return multierr.Append(c.primary.Sync(), c.fallback.Sync())
}

func (c *fallbackOptimizedForWith) clone() *fallbackOptimizedForWith {
	return &fallbackOptimizedForWith{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		primary:      c.primary,
		fallback:     c.fallback,
	}
}
