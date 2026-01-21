package openfga

import (
	"context"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	openfga "github.com/openfga/go-sdk"
)

type NoopClient struct {
	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func NewNoopClient(tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *NoopClient {
	c := new(NoopClient)
	c.tracer = tracer
	c.monitor = monitor
	c.logger = logger
	return c
}

func (c *NoopClient) ListObjects(ctx context.Context, user string, relation string, objectType string) ([]string, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.NoopClient.ListObjects")
	defer span.End()

	return make([]string, 0), nil
}

func (c *NoopClient) Check(ctx context.Context, user string, relation string, object string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.NoopClient.Check")
	defer span.End()

	return true, nil
}

func (c *NoopClient) ReadModel(ctx context.Context) (*openfga.AuthorizationModel, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.NoopClient.ReadModel")
	defer span.End()

	return nil, nil
}

func (c *NoopClient) WriteModel(ctx context.Context, model []byte) (string, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.NoopClient.WriteModel")
	defer span.End()

	return "", nil
}

func (c *NoopClient) CompareModel(ctx context.Context, model openfga.AuthorizationModel) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "openfga.NoopClient.CompareModel")
	defer span.End()

	return true, nil
}
