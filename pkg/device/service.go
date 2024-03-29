package device

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

type Service struct {
	hydra HydraClientInterface

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) AcceptUserCode(ctx context.Context, deviceChallenge string, req *hydra.AcceptDeviceUserCodeRequest) (*hClient.OAuth2RedirectTo, error) {
	ctx, span := s.tracer.Start(ctx, "device.service.AcceptUserCode")
	defer span.End()

	accept, res, err := s.hydra.OAuth2Api().AcceptUserCodeRequest(ctx).
		DeviceChallenge(deviceChallenge).
		AcceptDeviceUserCodeRequest(*req).
		Execute()

	if err != nil {
		s.logger.Debugf("full HTTP response: %v", res)
		return nil, err
	}

	return accept, nil
}

func (s *Service) ParseUserCodeBody(r *http.Request) (*hydra.AcceptDeviceUserCodeRequest, error) {
	body := new(hydra.AcceptDeviceUserCodeRequest)

	err := parseBody(r.Body, &body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func parseBody(b io.ReadCloser, body interface{}) error {
	decoder := json.NewDecoder(b)
	err := decoder.Decode(body)
	return err
}

func NewService(hydra HydraClientInterface, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.hydra = hydra

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}
