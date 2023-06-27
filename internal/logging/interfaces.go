package logging

type LoggerInterface interface {
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Debugf(string, ...interface{})
	Fatalf(string, ...interface{})
	Error(...interface{})
	Info(...interface{})
	Warn(...interface{})
	Debug(...interface{})
	Fatal(...interface{})
}
