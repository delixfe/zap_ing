package appender

import (
	"errors"
	"go.uber.org/zap/zapcore"
)

var ErrFailEnabled = errors.New("failing appender is failing")

var _ Appender = &Failing{}

type Failing struct {
	primary Appender
	fail    bool
}

func NewFailing(inner Appender, failing bool) *Failing {
	return &Failing{
		primary: inner,
		fail:    failing,
	}
}

func (a *Failing) Failing() bool {
	return a.fail
}

func (a *Failing) Fail() {
	a.fail = true
}

func (a *Failing) Proceed() {
	a.fail = false
}

func (a *Failing) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	if a.fail {
		return 0, ErrFailEnabled
	}
	n, err = a.primary.Write(p, ent)
	if err == nil {
		return
	}
	return

}

func (a *Failing) Sync() error {
	return a.Sync()
}
