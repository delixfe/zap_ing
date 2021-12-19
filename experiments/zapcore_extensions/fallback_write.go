package zapcore_extensions

import (
	"go.uber.org/multierr"
	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = &fallbackOptimizedForWrite{}

// With will have double the cost it normally would have
// Write will have no additional cost on the happy path
type fallbackOptimizedForWrite struct {
	zapcore.LevelEnabler

	// check will not be called
	// level will be ignored
	primary zapcore.Core

	fallback zapcore.Core
}

func (c *fallbackOptimizedForWrite) With(fields []zapcore.Field) zapcore.Core {
	primary := c.primary.With(fields)
	fallback := c.fallback.With(fields)
	return &fallbackOptimizedForWrite{
		LevelEnabler: c.LevelEnabler,
		primary:      primary,
		fallback:     fallback,
	}
}

func (c *fallbackOptimizedForWrite) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}
func (c *fallbackOptimizedForWrite) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	primErr := c.primary.Write(ent, fields)
	if primErr == nil {
		return nil
	}
	fallErr := c.fallback.Write(ent, fields)
	if fallErr == nil {
		return nil
	}

	// TODO: decide which error to return
	return multierr.Append(primErr, fallErr)
}

func (c *fallbackOptimizedForWrite) Sync() error {
	return multierr.Append(c.primary.Sync(), c.fallback.Sync())
}
