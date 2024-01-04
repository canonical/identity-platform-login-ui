package logging

type NoopLogger struct {
}

func NewNoopLogger() *NoopLogger {
	return new(NoopLogger)
}

func (l *NoopLogger) Errorf(string, ...interface{}) {}
func (l *NoopLogger) Infof(string, ...interface{})  {}
func (l *NoopLogger) Warnf(string, ...interface{})  {}
func (l *NoopLogger) Debugf(string, ...interface{}) {}
func (l *NoopLogger) Fatalf(string, ...interface{}) {}
func (l *NoopLogger) Error(...interface{})          {}
func (l *NoopLogger) Info(...interface{})           {}
func (l *NoopLogger) Warn(...interface{})           {}
func (l *NoopLogger) Debug(...interface{})          {}
func (l *NoopLogger) Fatal(...interface{})          {}
