package zapcore_extensions

import "go.uber.org/zap/zapcore"

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}
