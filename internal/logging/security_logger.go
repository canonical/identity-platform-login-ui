package logging

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const APP_ID = "identity_platform.login_ui"

var _ SecurityLoggerInterface = (*SecurityLogger)(nil)

type SecurityLogger struct {
	l *zap.Logger
}

type Field = zap.Field
type Option []Field

// We are mapping DPanicLevel to CRITICAL as zap does not have a Critical level
// This is a workaround to ensure that we comply to the Canonical OWASP logging
// recommendations. Otherwise we could have created a custom level, but it would have to
// to mapped to an integer below DEBUG level.
func levelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DPanicLevel:
		enc.AppendString("CRITICAL")
	default:
		enc.AppendString(l.CapitalString())
	}
}

func WithRequest(request *http.Request) Option {
	h := strings.Split(request.Host, ":")
	fields := []Field{
		zap.String("useragent", request.UserAgent()),
		zap.String("source_ip", request.RemoteAddr),
		zap.String("hostname", h[0]),
		zap.String("protocol", request.Proto),
		zap.String("request_uri", request.RequestURI),
		zap.String("request_method", request.Method),
	}
	if len(h) > 1 {
		fields = append(fields, zap.String("port", h[1]))
	}

	return fields
}

func WithContext(ctx context.Context) Option {
	if ctx == nil {
		return []Field{}
	}
	ret := []Field{}
	for key, label := range map[string]string{
		UserAgentKey:     "useragent",
		SourceIpKey:      "source_ip",
		HostnameKey:      "hostname",
		ProtocolKey:      "protocol",
		PortKey:          "port",
		RequestUriKey:    "request_uri",
		RequestMethodKey: "request_method",
	} {
		if v := ctx.Value(key); v != nil {
			ret = append(ret, zap.String(label, v.(string)))
		}
	}

	return ret
}

func WithLabel(key, value string) Option {
	return []Field{zap.String(key, value)}
}

func (a *SecurityLogger) SuccessfulLogin(user string, options ...Option) {
	msg := fmt.Sprintf("User %s login successfully", user)
	fields := []Field{zap.String("event", "authn_login_success:"+user)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info(msg, fields...)
}

func (a *SecurityLogger) FailedLogin(err string, options ...Option) {
	msg := "User login failed, " + err
	fields := []Field{zap.String("event", "authn_login_fail:"+err)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) AccountLockout(user string, options ...Option) {
	msg := fmt.Sprintf("User %s login locked because maxretries exceeded", user)
	fields := []Field{zap.String("event", fmt.Sprintf("authn_login_lock:%s,maxretries", user))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) PasswordChange(user string, options ...Option) {
	msg := fmt.Sprintf("User %s has successfully changed their password", user)
	fields := []Field{zap.String("event", "authn_password_change:"+user)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info(msg, fields...)
}

func (a *SecurityLogger) PasswordChangeFail(user string, options ...Option) {
	msg := fmt.Sprintf("User %s failed to change their password", user)
	fields := []Field{zap.String("event", "authn_password_change_fail:"+user)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}

	a.l.DPanic(msg, fields...)
}

func (a *SecurityLogger) TokenCreate(options ...Option) {
	fields := []Field{zap.String("event", "authn_token_created:"+APP_ID)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info("A token has been created", fields...)
}

func (a *SecurityLogger) TokenRevoke(options ...Option) {
	fields := []Field{zap.String("event", "authn_token_revoked:"+APP_ID)}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info("A token has been revoked", fields...)
}

func (a *SecurityLogger) TokenReuse(token string, options ...Option) {
	msg := fmt.Sprintf("Someone attempted to use token ID: %s which was previously revoked", token)
	fields := []Field{zap.String("event", fmt.Sprintf("authn_token_reuse:%s,%s", APP_ID, token))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.DPanic(msg, fields...)
}

func (a *SecurityLogger) TokenDelete(user string, options ...Option) {
	msg := "Session was deleted for user " + user
	fields := []Field{zap.String("event", fmt.Sprintf("authn_token_delete:%s", user))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info(msg, fields...)
}

func (a *SecurityLogger) AuthzFailure(user, resource string, options ...Option) {
	msg := fmt.Sprintf("User %s attempted to access resource %s without entitlement", user, resource)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,%s", user, resource))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.DPanic(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureNotEmployee(user string, options ...Option) {
	msg := fmt.Sprintf("User %s logged in, but they are not a canonical employee", user)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,not_employee", user))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.DPanic(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureNoSession(api string, options ...Option) {
	msg := fmt.Sprintf("User tried to access the %s API without a session", api)
	fields := []Field{zap.String("event", "authz_fail:no_session")}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.DPanic(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureApplicationAccess(user, clientID string, options ...Option) {
	msg := fmt.Sprintf("User %s tried to access application %s", user, clientID)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,%s", user, clientID))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureInsufficientPermissions(user, action, api string, options ...Option) {
	msg := fmt.Sprintf("User %s tried to perform `%s` on the `%s` API without enough permissions", user, action, api)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,%s", user, api))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureRoleAssignment(user, roles string, options ...Option) {
	msg := fmt.Sprintf("User %s tried to assign the `%s` roles  without enough permissions", user, roles)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,%s", user, roles))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) AuthzFailureIdentityAssignment(user, identities string, options ...Option) {
	msg := fmt.Sprintf("User %s tried to assign the `%s` identities  without enough permissions", user, identities)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_fail:%s,identities", user))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn(msg, fields...)
}

func (a *SecurityLogger) AdminAction(user, action, api, resource string, options ...Option) {
	msg := fmt.Sprintf("User %s has %s the `%s` `%s`", user, action, api, resource)
	fields := []Field{zap.String("event", fmt.Sprintf("authz_admin:%s,%s,%s", user, api, action))}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Info(msg, fields...)
}

func (a *SecurityLogger) SystemStartup(options ...Option) {
	fields := []Field{zap.String("event", "system_startup")}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn("New instance spawned", fields...)
}

func (a *SecurityLogger) SystemShutdown(options ...Option) {
	fields := []Field{zap.String("event", "system_shutdown")}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn("Instance stopped", fields...)
}

func (a *SecurityLogger) SystemRestart(options ...Option) {
	fields := []Field{zap.String("event", "system_restart")}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn("Instance restarted", fields...)
}

func (a *SecurityLogger) SystemCrash(options ...Option) {
	fields := []Field{zap.String("event", "system_crash")}
	for _, opt := range options {
		fields = append(fields, opt...)
	}
	a.l.Warn("Instance crashed", fields...)
}

func (a *SecurityLogger) Sync() {
	a.l.Sync()
}

// NewSecurityLogger creates a new security logger instance
func NewSecurityLogger(l string) *SecurityLogger {
	var lvl zapcore.Level

	switch strings.ToLower(l) {
	case "debug":
		lvl = zap.DebugLevel
	case "info":
		lvl = zap.InfoLevel
	case "warn":
		lvl = zap.WarnLevel
	case "error":
		lvl = zap.ErrorLevel
	case "critical":
		lvl = zap.DPanicLevel
	}

	c := zapcore.EncoderConfig{
		MessageKey:  "description",
		LevelKey:    "level",
		EncodeLevel: levelEncoder,
		TimeKey:     "datetime",
		EncodeTime:  zapcore.RFC3339NanoTimeEncoder,
	}

	encoder := zapcore.NewJSONEncoder(c)
	encoder.AddString("appid", APP_ID)
	encoder.AddString("type", "security")
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(lvl))

	logger := new(SecurityLogger)
	logger.l = zap.New(core)

	return logger
}
