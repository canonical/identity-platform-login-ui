package status

import (
	"context"

	"github.com/canonical/identity-platform-login-ui/internal/healthcheck"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"go.opentelemetry.io/otel/trace"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

type Service struct {
	kratos kClient.MetadataApi
	hydra  hClient.MetadataApi

	hydraStatus  healthcheck.CheckerInterface
	kratosStatus healthcheck.CheckerInterface

	tracer  trace.Tracer
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) KratosStatus(ctx context.Context) bool {
	ctx, span := s.tracer.Start(ctx, "status.Service.KratosStatus")
	defer span.End()

	return s.kratosStatus.Status()
}

func (s *Service) HydraStatus(ctx context.Context) bool {
	ctx, span := s.tracer.Start(ctx, "status.Service.HydraStatus")
	defer span.End()

	return s.hydraStatus.Status()
}

func (s *Service) kratosReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.kratosReady")
	defer span.End()

	// IsReady only checks the status of specific instance called, not the cluster status
	ok, _, err := s.kratos.IsReady(ctx).Execute()

	var available float64

	if ok != nil && err == nil {
		available = 1.0
	}

	tags := map[string]string{"component": "kratos"}

	s.monitor.SetDependencyAvailability(tags, available)

	return ok != nil, err
}

func (s *Service) hydraReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.hydraReady")
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

func NewService(kratos kClient.MetadataApi, hydra hClient.MetadataApi, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.kratos = kratos
	s.hydra = hydra

	s.hydraStatus = healthcheck.NewChecker(s.hydraReady, tracer, logger)
	s.kratosStatus = healthcheck.NewChecker(s.kratosReady, tracer, logger)

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	// TOOO @shipperizer hook up the Stop methods for each checker
	s.hydraStatus.Start()
	s.kratosStatus.Start()

	return s
}
