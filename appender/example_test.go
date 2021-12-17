package appender_test

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
	"zap_ing/appender"
	"zap_ing/appender/chaos"
)

var encoderConfig = zapcore.EncoderConfig{
	MessageKey:       "msg",
	LevelKey:         "level",
	NameKey:          "logger",
	EncodeLevel:      zapcore.LowercaseLevelEncoder,
	EncodeTime:       zapcore.ISO8601TimeEncoder,
	EncodeDuration:   zapcore.StringDurationEncoder,
	ConsoleSeparator: " ** ",
}

func Example_core() {

	writer := appender.NewWriter(zapcore.Lock(os.Stdout))

	failing := chaos.NewFailingSwitchable(writer)

	// this could be a TcpWriter
	var primaryOut appender.Appender = failing

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := appender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := appender.NewFallback(primaryOut, secondaryOut)

	core := appender.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), fallback, zapcore.DebugLevel)
	logger := zap.New(core)

	logger.Info("zappig")

	failing.Break()

	logger.Info("on the fallback")

	// Output:
	// info ** zappig
	// FALLBACK: info ** on the fallback
}

func ExampleAsync() {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	writer := appender.NewWriter(zapcore.Lock(os.Stdout))

	failing := chaos.NewFailingSwitchable(writer)
	blocking := chaos.NewBlockingSwitchable(ctx, failing)

	// this could be a TcpWriter
	var primaryOut appender.Appender = blocking

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := appender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := appender.NewFallback(primaryOut, secondaryOut)
	async := appender.NewAsync(fallback, secondaryOut)

	core := appender.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), async, zapcore.DebugLevel)
	logger := zap.New(core)

	logger.Info("this logs async")

	blocking.Break()

	for i := 0; i < 1001; i++ {
		logger.Info("while blocked", zap.Int("i", i))
	}

	time.Sleep(time.Second)
	blocking.Fix()
	async.Drain(ctx)

	// Output:
	// info ** zappig
	// FALLBACK: info ** on the fallback
}
