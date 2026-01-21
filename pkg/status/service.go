package status

import (
	"context"
	"runtime/debug"

	"github.com/canonical/identity-platform-login-ui/internal/healthcheck"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/internal/version"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type BuildInfo struct {
	Version    string `json:"version"`
	CommitHash string `json:"commit_hash"`
	Name       string `json:"name"`
}

type Service struct {
	kratos kClient.MetadataAPI
	hydra  hClient.MetadataAPI

	hydraStatus  healthcheck.CheckerInterface
	kratosStatus healthcheck.CheckerInterface

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) BuildInfo(ctx context.Context) *BuildInfo {
	ctx, span := s.tracer.Start(ctx, "status.Service.BuildInfo")
	defer span.End()

	info, ok := debug.ReadBuildInfo()

	if !ok {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	buildInfo := new(BuildInfo)
	buildInfo.Name = info.Main.Path
	buildInfo.Version = version.Version
	buildInfo.CommitHash = s.gitRevision(ctx, info.Settings)

	span.SetStatus(codes.Ok, "")
	return buildInfo
}

func (s *Service) KratosStatus(ctx context.Context) bool {
	ctx, span := s.tracer.Start(ctx, "status.Service.KratosStatus")
	defer span.End()

	span.SetStatus(codes.Ok, "")
	return s.kratosStatus.Status()
}

func (s *Service) HydraStatus(ctx context.Context) bool {
	ctx, span := s.tracer.Start(ctx, "status.Service.HydraStatus")
	defer span.End()

	span.SetStatus(codes.Ok, "")
	return s.hydraStatus.Status()
}

func (s *Service) kratosReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.kratosReady")
	defer span.End()

	// IsReady only checks the status of specific instance called, not the cluster status
	ok, resp, err := s.kratos.IsReady(ctx).Execute()

	if resp != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	}

	var available float64

	if ok != nil && err == nil {
		available = 1.0
	}

	tags := map[string]string{"component": "kratos"}

	s.monitor.SetDependencyAvailability(tags, available)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return ok != nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ok != nil, err
}

func (s *Service) hydraReady(ctx context.Context) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "status.Service.hydraReady")
	defer span.End()

	// IsReady only checks the status of specific instance called, not the cluster status
	ok, resp, err := s.hydra.IsReady(ctx).Execute()

	if resp != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", resp.StatusCode))
	}

	var available float64

	if ok != nil && err == nil {
		available = 1.0
	}

	tags := map[string]string{"component": "hydra"}

	s.monitor.SetDependencyAvailability(tags, available)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return ok != nil, err
	}

	span.SetStatus(codes.Ok, "")
	return ok != nil, err
}

func (s *Service) gitRevision(ctx context.Context, settings []debug.BuildSetting) string {
	ctx, span := s.tracer.Start(ctx, "status.Service.gitRevision")
	defer span.End()

	for _, setting := range settings {
		if setting.Key == "vcs.revision" {
			span.SetStatus(codes.Ok, "")
			return setting.Value
		}
	}

	span.SetStatus(codes.Ok, "")
	return "n/a"
}

func NewService(kratos kClient.MetadataAPI, hydra hClient.MetadataAPI, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
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
