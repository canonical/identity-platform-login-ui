package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	misc "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

type Service struct {
	hydra HydraClientInterface

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) GetConsent(ctx context.Context, challenge string) (*hClient.OAuth2ConsentRequest, error) {
	ctx, span := s.tracer.Start(ctx, "hydra.OAuth2API.GetOAuth2ConsentRequest")
	defer span.End()

	consent, res, err := s.hydra.OAuth2API().GetOAuth2ConsentRequest(
		ctx,
	).ConsentChallenge(challenge).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		// TODO @shipperizer we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return consent, nil
}

func (s *Service) AcceptConsent(ctx context.Context, identity kClient.Identity, consent *hClient.OAuth2ConsentRequest) (*hClient.OAuth2RedirectTo, error) {
	session := hClient.NewAcceptOAuth2ConsentRequestSession()
	session.SetIdToken(misc.GetUserClaims(identity, *consent))

	r := hClient.NewAcceptOAuth2ConsentRequest()
	r.SetGrantScope(consent.RequestedScope)
	r.SetGrantAccessTokenAudience(consent.RequestedAccessTokenAudience)
	r.SetSession(*session)
	r.SetRemember(true)

	ctx, span := s.tracer.Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest")
	defer span.End()

	accept, res, err := s.hydra.OAuth2API().AcceptOAuth2ConsentRequest(
		ctx,
	).ConsentChallenge(
		consent.GetChallenge(),
	).AcceptOAuth2ConsentRequest(
		*r,
	).Execute()
	if res != nil {
		span.SetAttributes(attribute.Int("http.response.status_code", res.StatusCode))
	}

	if err != nil {
		// TODO @shipperizer we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", res)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return accept, nil
}

func NewService(hydra HydraClientInterface, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.hydra = hydra

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}
