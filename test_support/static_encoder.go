package test_support

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"time"
	"zap_ing/internal/bufferpool"
)

func NewStaticEncoder(returnValue []byte) *staticEncoder {
	buf := bufferpool.Get()
	_, _ = buf.Write(returnValue)
	return &staticEncoder{buf: buf}
}

type staticEncoder struct {
	buf *buffer.Buffer
}

func (e *staticEncoder) AddArray(_ string, _ zapcore.ArrayMarshaler) error { return nil }

func (e *staticEncoder) AddObject(_ string, _ zapcore.ObjectMarshaler) error { return nil }

func (e *staticEncoder) AddBinary(_ string, _ []byte) {}

func (e *staticEncoder) AddByteString(_ string, _ []byte) {}

func (e *staticEncoder) AddBool(_ string, _ bool) {}

func (e *staticEncoder) AddComplex128(_ string, _ complex128) {}

func (e *staticEncoder) AddComplex64(_ string, _ complex64) {}

func (e *staticEncoder) AddDuration(_ string, _ time.Duration) {}

func (e *staticEncoder) AddFloat64(_ string, _ float64) {}

func (e *staticEncoder) AddFloat32(_ string, _ float32) {}

func (e *staticEncoder) AddInt(_ string, _ int) {}

func (e *staticEncoder) AddInt64(_ string, _ int64) {}

func (e *staticEncoder) AddInt32(_ string, _ int32) {}

func (e *staticEncoder) AddInt16(_ string, _ int16) {}

func (e *staticEncoder) AddInt8(_ string, _ int8) {}

func (e *staticEncoder) AddString(_, _ string) {}

func (e *staticEncoder) AddTime(_ string, _ time.Time) {}

func (e *staticEncoder) AddUint(_ string, _ uint) {}

func (e *staticEncoder) AddUint64(_ string, _ uint64) {}

func (e *staticEncoder) AddUint32(_ string, _ uint32) {}

func (e *staticEncoder) AddUint16(_ string, _ uint16) {}

func (e *staticEncoder) AddUint8(_ string, _ uint8) {}

func (e *staticEncoder) AddUintptr(_ string, _ uintptr) {}

func (e *staticEncoder) AddReflected(_ string, _ interface{}) error { return nil }

func (e *staticEncoder) OpenNamespace(_ string) {}

func (e *staticEncoder) Clone() zapcore.Encoder {
	return e.clone()
}

func (e *staticEncoder) clone() *staticEncoder {
	buf := bufferpool.Get()
	_, _ = buf.Write(e.buf.Bytes())
	return &staticEncoder{buf: buf}
}

func (e *staticEncoder) EncodeEntry(_ zapcore.Entry, _ []zapcore.Field) (*buffer.Buffer, error) {
	return e.clone().buf, nil
}
