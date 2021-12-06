package zap_ing

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"

	"zap_ing/internal/bufferpool"
)

type EnvelopingFn func(ent *zapcore.Entry, encoded *buffer.Buffer, output *buffer.Buffer) error

// NewEnveloping creates a Core that duplicates log entries into two or more
// underlying Cores.
//
// Calling it with a single Core returns the input unchanged, and calling
// it with no input returns a no-op Core.
func NewEnveloping(enc zapcore.Encoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler, envFn EnvelopingFn) zapcore.Core {
	//creates a Core that writes logs to a WriteSyncer.
	return &envelopingCore{
		LevelEnabler: enab,
		enc:          enc,
		out:          ws,
		envFn:        envFn,
	}
}

type envelopingCore struct {
	zapcore.LevelEnabler
	enc   zapcore.Encoder
	out   zapcore.WriteSyncer
	envFn EnvelopingFn
}

func (c *envelopingCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	addFields(clone.enc, fields)
	return clone
}

func (c *envelopingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *envelopingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	encoded, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	enveloped := bufferpool.Get()
	err = c.envFn(&ent, encoded, enveloped)
	encoded.Free()
	if err != nil {
		return err
	}

	_, err = c.out.Write(enveloped.Bytes())
	enveloped.Free()
	if err != nil {
		return err
	}
	if ent.Level > zapcore.ErrorLevel {
		// Since we may be crashing the program, sync the output. Ignore Sync
		// errors, pending a clean solution to issue #370.
		c.Sync()
	}
	return nil
}

func (c *envelopingCore) Sync() error {
	return c.out.Sync()
}

func (c *envelopingCore) clone() *envelopingCore {
	return &envelopingCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
	}
}
