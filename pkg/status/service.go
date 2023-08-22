package status

import (
	"context"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"go.opentelemetry.io/otel/trace"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

type Service struct {
	kratos kClient.MetadataApi
	hydra  hClient.MetadataApi

	tracer  trace.Tracer
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) CheckKratosReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.CheckKratosReady")
	defer span.End()

	ok, _, err := s.kratos.IsReady(ctx).Execute()

	var available float64

	if ok != nil && err == nil {
		available = 1.0
	}

	tags := map[string]string{"component": "kratos"}

	s.monitor.SetDependencyAvailability(tags, available)

	return ok != nil, err
}

func (s *Service) CheckHydraReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.CheckHydraReady")
	defer span.End()

	// IsReady only checks the status of specific instance called, not the cluster status
	ok, _, err := s.hydra.IsReady(ctx).Execute()

	var available float64

	if ok != nil && err == nil {
		available = 1.0
	}

	tags := map[string]string{"component": "hydra"}

	s.monitor.SetDependencyAvailability(tags, available)

	return ok != nil, err
}

func NewService(kmeta kClient.MetadataApi, hmeta hClient.MetadataApi, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.kratos = kmeta
	s.hydra = hmeta

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}
