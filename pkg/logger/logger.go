package logger

import log "github.com/sirupsen/logrus"

type Leveled struct {
	*log.Logger
}

func (l Leveled) withFields(keysAndValues []interface{}) *log.Entry {
	f := make(map[string]interface{})

	for i := 0; i < len(keysAndValues)-1; i += 2 {
		f[keysAndValues[i].(string)] = keysAndValues[i+1]
	}

	return l.WithFields(f)
}

func (l Leveled) Error(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Error(msg)
}

func (l Leveled) Info(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Info(msg)
}
func (l Leveled) Debug(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Debug(msg)
}

func (l Leveled) Warn(msg string, keysAndValues ...interface{}) {
	l.withFields(keysAndValues).Warn(msg)
}
