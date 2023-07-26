package tracing

import (
	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

type Config struct {
	OtelHTTPEndpoint string
	OtelGRPCEndpoint string
	Logger           logging.LoggerInterface

	Enabled bool
}

func NewConfig(enabled bool, otelGRPCEndpoint, otelHTTPEndpoint string, logger logging.LoggerInterface) *Config {
	c := new(Config)

	c.OtelGRPCEndpoint = otelGRPCEndpoint
	c.OtelHTTPEndpoint = otelHTTPEndpoint
	c.Logger = logger
	c.Enabled = enabled

	return c
}
