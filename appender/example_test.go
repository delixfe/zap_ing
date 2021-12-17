package appender_test

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"os"
	"zap_ing/appender"
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

	fallbackOut := appender.NewEnveloping(writer, func(p []byte, ent *zapcore.Entry, output *buffer.Buffer) error {
		output.WriteString("FALLBACK: ")
		output.Write(p)
		return nil
	})

	// this could be a TcpWriter
	primaryOut := appender.NewFailing(writer, false)

	// this would normally be os.Stdout or Stderr without further wrapping
	fallback := appender.NewFallback(primaryOut, fallbackOut)

	core := appender.NewAppenderCore(zapcore.NewConsoleEncoder(encoderConfig), fallback, zapcore.DebugLevel)

	logger := zap.New(core)

	logger.Info("zappig")

	primaryOut.Fail()

	logger.Info("on the fallback")

	// Output:
	// info ** zappig
	// FALLBACK: info ** on the fallback
}
