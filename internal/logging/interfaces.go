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
	Security() SecurityLoggerInterface
}

type SecurityLoggerInterface interface {
	SuccessfulLogin(string, ...Option)
	FailedLogin(string, ...Option)
	AccountLockout(string, ...Option)
	PasswordChange(string, ...Option)
	PasswordChangeFail(string, ...Option)
	TokenCreate(...Option)
	TokenRevoke(...Option)
	TokenReuse(string, ...Option)
	TokenDelete(string, ...Option)
	AdminAction(string, string, string, string, ...Option)
	AuthzFailure(string, string, ...Option)
	AuthzFailureNotEmployee(string, ...Option)
	AuthzFailureApplicationAccess(string, string, ...Option)
	AuthzFailureNoSession(string, ...Option)
	AuthzFailureInsufficientPermissions(string, string, string, ...Option)
	AuthzFailureRoleAssignment(string, string, ...Option)
	AuthzFailureIdentityAssignment(string, string, ...Option)
	SystemStartup(...Option)
	SystemShutdown(...Option)
	SystemRestart(...Option)
	SystemCrash(...Option)
}
