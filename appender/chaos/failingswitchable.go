package chaos

import (
	"errors"
	"github.com/delixfe/zap_ing/appender"
	"go.uber.org/zap/zapcore"
)

var ErrFailEnabled = errors.New("failing switchable appender is failing")

var (
	_ appender.Appender = &FailingSwitchable{}
	_ Switchable        = &FailingSwitchable{}
)

type FailingSwitchable struct {
	primary appender.Appender
	enabled bool
}

func NewFailingSwitchable(inner appender.Appender) *FailingSwitchable {
	return &FailingSwitchable{
		primary: inner,
		enabled: false,
	}
}

func (a *FailingSwitchable) Breaking() bool {
	return a.enabled
}

func (a *FailingSwitchable) Break() {
	a.enabled = true
}

func (a *FailingSwitchable) Fix() {
	a.enabled = false
}

func (a *FailingSwitchable) Write(p []byte, ent zapcore.Entry) (n int, err error) {
	if a.enabled {
		return 0, ErrFailEnabled
	}
	n, err = a.primary.Write(p, ent)
	if err == nil {
		return
	}
	return

}

func (a *FailingSwitchable) Sync() error {
	return a.Sync()
}
