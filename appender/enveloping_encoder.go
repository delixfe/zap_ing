package appender

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"zap_ing/internal/bufferpool"
)

// envelopingEncoder decorates zapcore.Encoder
type envelopingEncoder struct {
	inner zapcore.Encoder
	envFn EnvelopingFn
}

func (r *envelopingEncoder) Clone() zapcore.Encoder {
	return &envelopingEncoder{
		inner: r.inner.Clone(),
		envFn: r.envFn,
	}
}

func (r *envelopingEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	encoded, err := r.inner.EncodeEntry(ent, fields)
	if err != nil {
		return nil, err
	}
	enveloped := bufferpool.Get()
	err = r.envFn(&ent, encoded, enveloped)
	encoded.Free()
	if err != nil {
		return nil, err
	}
	return enveloped, nil
}
