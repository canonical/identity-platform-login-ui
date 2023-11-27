package openfga

import (
	"context"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	openfga "github.com/openfga/go-sdk"
	"go.opentelemetry.io/otel/trace"
)

type NoopClient struct {
	tracer  trace.Tracer
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func NewNoopClient(tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *NoopClient {
	c := new(NoopClient)
	c.tracer = tracer
	c.monitor = monitor
	c.logger = logger
	return c
}

func (c *NoopClient) ListObjects(ctx context.Context, user string, relation string, objectType string) ([]string, error) {
	return make([]string, 0), nil
}

func (c *NoopClient) Check(ctx context.Context, user string, relation string, object string) (bool, error) {
	return true, nil
}

func (c *NoopClient) ReadModel(ctx context.Context) (*openfga.AuthorizationModel, error) {
	return nil, nil
}

func (c *NoopClient) CompareModel(ctx context.Context, model openfga.AuthorizationModel) (bool, error) {
	return true, nil
}
