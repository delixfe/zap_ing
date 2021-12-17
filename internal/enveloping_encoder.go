package internal

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"zap_ing/appender"
	"zap_ing/internal/bufferpool"
)

// TODO: extract the decorator approach and remove

// envelopingEncoder decorates zapcore.Encoder
type envelopingEncoder struct {
	inner zapcore.Encoder
	envFn appender.EnvelopingFn
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
