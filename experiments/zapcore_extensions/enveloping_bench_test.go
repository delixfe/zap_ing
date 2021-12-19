package zapcore_extensions

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"io"
	"math/rand"
	"testing"
	"zap_ing/test_support"
)

func BenchmarkNewEnveloping(b *testing.B) {

	random := rand.New(rand.NewSource(0))

	length := map[string]int{
		"min": 559,
		"p50": 776,
		"p95": 1081,
		"max": 5397,
	}

	for lengthKey, lengthValue := range length {
		encodedBytes := make([]byte, lengthValue)
		_, _ = random.Read(encodedBytes)
		encodedBytes[lengthValue-1] = '\n'

		staticEncoder := test_support.NewStaticEncoder(encodedBytes)

		run := func(b *testing.B, core zapcore.Core) {
			logger := zap.New(core)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				logger.Info("")
			}
		}

		b.Run("bare_"+lengthKey, func(b *testing.B) {
			writer := zapcore.AddSync(io.Discard)
			core := zapcore.NewCore(staticEncoder, writer, zapcore.DebugLevel)
			run(b, core)
		})

		b.Run("enveloping_"+lengthKey, func(b *testing.B) {
			syslogPrefix := "<13>2021-12-06T20:40:13.724+01:00 11111111-e42f-4099-4e87-a0da doppler[0]: {"
			envFn := func(ent *zapcore.Entry, encoded *buffer.Buffer, output *buffer.Buffer) error {
				encoded.TrimNewline()
				output.AppendString(syslogPrefix)
				output.AppendString(ent.LoggerName)
				_, _ = output.Write(encoded.Bytes())
				output.AppendByte('}')
				output.AppendByte('\n')
				return nil
			}
			writer := zapcore.AddSync(io.Discard)
			core := NewEnveloping(staticEncoder, writer, zapcore.DebugLevel, envFn)
			run(b, core)
		})
	}
}
