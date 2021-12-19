package appender_test

import (
	"context"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
	"zap_ing/appender"
	"zap_ing/appender/appendercore"
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
	var primaryOut appendercore.Appender = failing

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := appender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := appender.NewFallback(primaryOut, secondaryOut)

	core := appendercore.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), fallback, zapcore.DebugLevel)
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
	blocking := chaos.NewBlockingSwitchableCtx(ctx, failing)

	// this could be a TcpWriter
	var primaryOut appendercore.Appender = appender.NewEnvelopingPreSuffix(blocking, "PRIMARY:  ", "")

	// this would normally be os.Stdout or Stderr without further wrapping
	secondaryOut := appender.NewEnvelopingPreSuffix(writer, "FALLBACK: ", "")

	fallback := appender.NewFallback(primaryOut, secondaryOut)
	async, _ := appender.NewAsync(fallback,
		appender.AsyncOnQueueNearlyFullForwardTo(appender.NewEnvelopingPreSuffix(writer, "QFALLBACK: ", "")),
		appender.AsyncMaxQueueLength(10),
		appender.AsyncQueueMinFreePercent(0.2),
		appender.AsyncQueueMonitorPeriod(time.Millisecond),
	)

	core := appendercore.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), async, zapcore.DebugLevel)
	logger := zap.New(core)

	logger.Info("this logs async")

	time.Sleep(time.Millisecond * 10)

	blocking.Break()

	logger.Info("primary blocks while trying to send this", zap.Int("i", 1))
	for i := 2; i <= 15; i++ {
		logger.Info("while broken", zap.Int("i", i))
	}

	blocking.Fix()
	time.Sleep(time.Millisecond * 10)
	async.Drain(ctx)

	// Output:
	// info ** zappig
	// FALLBACK: info ** on the fallback
}
