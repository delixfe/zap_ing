package appender_test

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"io"
	"testing"
	"zap_ing/appender"
	"zap_ing/internal/bufferpool"
)

func BenchmarkFallbackEnveloping(b *testing.B) {

	writer := appender.NewWriter(zapcore.AddSync(io.Discard))

	b.Run("no appender", func(b *testing.B) {
		core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(io.Discard), zapcore.DebugLevel)
		RunWithCore(core, b)
	})
	b.Run("writer", func(b *testing.B) {
		RunWithAppender(writer, b)
	})
	b.Run("fallback", func(b *testing.B) {
		a := appender.NewFallback(writer, writer)
		RunWithAppender(a, b)
	})
	b.Run("enveloping empty", func(b *testing.B) {
		envFnEmpty := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
			return nil
		}
		a := appender.NewEnveloping(writer, envFnEmpty)
		RunWithAppender(a, b)
	})
	b.Run("enveloping id", func(b *testing.B) {
		envId := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
			// write content from orig buffer in new buffer
			_, _ = output.Write(p)
			return nil
		}
		a := appender.NewEnveloping(writer, envId)
		RunWithAppender(a, b)
	})
	b.Run("enveloping prefix", func(b *testing.B) {
		envFn := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
			output.WriteString("prefix: ")
			_, _ = output.Write(p)
			return nil
		}
		a := appender.NewEnveloping(writer, envFn)
		RunWithAppender(a, b)
	})
}

func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := bufferpool.Get()
		buf.Free()
	}
}

func RunWithAppender(a appender.Appender, b *testing.B) {
	core := appender.NewAppenderCore(zapcore.NewJSONEncoder(encoderConfig), a, zapcore.DebugLevel)
	RunWithCore(core, b)
}

func RunWithCore(core zapcore.Core, b *testing.B) {
	logger := zap.New(core)
	logger.Info("Warmup")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("message")
	}
}
