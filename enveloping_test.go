package zap_ing

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"testing"
	"zap_ing/test_support"
)

func TestNewEnveloping(t *testing.T) {

	envFn := func(ent *zapcore.Entry, encoded *buffer.Buffer, output *buffer.Buffer) error {
		output.AppendString("START ")
		output.Write(encoded.Bytes())
		output.AppendString(" END")
		return nil
	}

	staticEncoder := test_support.NewStaticEncoder([]byte("static"))
	writer := test_support.Buffer{}

	core := NewEnveloping(staticEncoder, &writer, zapcore.DebugLevel, envFn)
	logger := zap.New(core)

	logger.Info("")

	assert.Equal(t, "START static END", writer.Stripped())
}
