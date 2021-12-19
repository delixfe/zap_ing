package appender_test

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"io"
	"strings"
	"testing"
	"time"
	"zap_ing/appender"
	"zap_ing/appender/appendercore"
)

type benchConfig struct {
	message string
}

func BenchmarkFallbackEnveloping(b *testing.B) {
	tests := []struct {
		name   string
		config benchConfig
	}{
		{name: "short message", config: benchConfig{message: "message"}},
		{name: "long message", config: benchConfig{message: strings.Repeat("x", 1000)}},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {

			writer := appender.NewWriter(zapcore.AddSync(io.Discard))
			config := tt.config
			b.Run("no appender", func(b *testing.B) {
				core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(io.Discard), zapcore.DebugLevel)
				RunWithCore(core, b, config)
			})
			b.Run("writer", func(b *testing.B) {
				RunWithAppender(writer, b, config)
			})
			b.Run("fallback", func(b *testing.B) {
				a := appender.NewFallback(writer, writer)
				RunWithAppender(a, b, config)
			})
			b.Run("enveloping empty", func(b *testing.B) {
				envFnEmpty := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
					return nil
				}
				a := appender.NewEnveloping(writer, envFnEmpty)
				RunWithAppender(a, b, config)
			})
			b.Run("enveloping id", func(b *testing.B) {
				envId := func(p []byte, ent zapcore.Entry, output *buffer.Buffer) error {
					// write content from orig buffer in new buffer
					_, _ = output.Write(p)
					return nil
				}
				a := appender.NewEnveloping(writer, envId)
				RunWithAppender(a, b, config)
			})
			b.Run("enveloping prefix", func(b *testing.B) {
				a := appender.NewEnvelopingPreSuffix(writer, "prefix: ", "")
				RunWithAppender(a, b, config)
			})
			b.Run("async", func(b *testing.B) {
				a, _ := appender.NewAsync(writer,
					appender.AsyncMaxQueueLength(1000),
					appender.AsyncQueueMonitorPeriod(time.Hour),
				)
				b.Cleanup(func() {
					a.Shutdown(nil)
				})
				RunWithAppender(a, b, config)
			})
			b.Run("chained_no_async", func(b *testing.B) {
				var a appendercore.Appender = writer
				a = appender.NewEnvelopingPreSuffix(a, "prefix: ", "")
				a = appender.NewFallback(a, writer)
				RunWithAppender(a, b, config)
			})
		})
	}
}

func RunWithAppender(a appendercore.Appender, b *testing.B, config benchConfig) {
	core := appendercore.NewAppenderCore(zapcore.NewJSONEncoder(encoderConfig), a, zapcore.DebugLevel)
	RunWithCore(core, b, config)
}

func RunWithCore(core zapcore.Core, b *testing.B, config benchConfig) {
	message := config.message
	logger := zap.New(core)
	logger.Info("Warmup")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info(message)
	}
}
