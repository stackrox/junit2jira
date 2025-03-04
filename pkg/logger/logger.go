package logger

import (
	"github.com/hashicorp/go-retryablehttp"
	log "github.com/sirupsen/logrus"
)

type leveled struct {
	l *log.Logger
}

func NewLeveled() retryablehttp.LeveledLogger {
	return leveled{l: log.StandardLogger()}
}

func (l leveled) withFields(keysAndValues []interface{}) *log.Entry {
	f := make(map[string]interface{})

	for i := 0; i < len(keysAndValues)-1; i += 2 {
		f[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	return l.l.WithFields(f)
}

func (l leveled) Error(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Error(msg)
}

func (l leveled) Info(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Info(msg)
}
func (l leveled) Debug(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Debug(msg)
}

func (l leveled) Warn(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Warn(msg)
}
