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

	_, r, err := s.kratos.IsReady(ctx).Execute()

	if err != nil {
		s.logger.Error(err)
		s.logger.Debugf("full HTTP response: %v", r)

	}

	return err == nil, err

}

func (s *Service) CheckHydraReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.CheckHydraReady")
	defer span.End()

	_, r, err := s.hydra.IsReady(ctx).Execute()

	if err != nil {
		s.logger.Error(err)
		s.logger.Debugf("full HTTP response: %v", r)
	}

	return err == nil, err

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
