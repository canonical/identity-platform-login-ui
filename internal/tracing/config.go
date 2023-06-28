package tracing

import (
	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

type Config struct {
	JaegerEndpoint string
	Logger         logging.LoggerInterface

	Enabled bool
}

func NewConfig(enabled bool, endpoint string, logger logging.LoggerInterface) *Config {
	c := new(Config)

	c.JaegerEndpoint = endpoint
	c.Logger = logger
	c.Enabled = enabled

	return c
}
