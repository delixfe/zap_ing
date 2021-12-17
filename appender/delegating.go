package appender

import "go.uber.org/zap/zapcore"

var _ Appender = &Writer{}

type Delegating struct {
	WriteFn func(p []byte, ent zapcore.Entry) (n int, err error)
	SyncFn  func() error
}

func NewDelegating(writeFn func(p []byte, ent zapcore.Entry) (n int, err error), syncFn func() error) *Delegating {
	return &Delegating{
		WriteFn: writeFn,
		SyncFn:  syncFn,
	}
}

func (a *Delegating) Write(p []byte, ent zapcore.Entry) (int, error) {
	writeFn := a.WriteFn
	if writeFn == nil {
		return len(p), nil
	}
	return writeFn(p, ent)
}

func (a *Delegating) Sync() error {
	syncFn := a.SyncFn
	if syncFn == nil {
		return nil
	}
	return syncFn()
}
