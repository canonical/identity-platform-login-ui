package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"

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
	ctx, span := s.tracer.Start(ctx, "hydra.OAuth2Api.GetOAuth2ConsentRequest")
	defer span.End()

	consent, res, err := s.hydra.OAuth2Api().GetOAuth2ConsentRequest(
		ctx,
	).ConsentChallenge(challenge).Execute()

	if err != nil {
		// TODO @shipperizer we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", res)

		return nil, err
	}

	return consent, nil
}

func (s *Service) AcceptConsent(ctx context.Context, identity kClient.Identity, consent *hClient.OAuth2ConsentRequest) (*hClient.OAuth2RedirectTo, error) {
	session := hClient.NewAcceptOAuth2ConsentRequestSession()
	session.SetIdToken(misc.GetUserClaims(identity, *consent))

	atAudience := make([]string, 0)
	if consent.RequestedAccessTokenAudience != nil {
		atAudience = append(atAudience, consent.RequestedAccessTokenAudience...)
	}
	if consent.HasClient() {
		atAudience = append(atAudience, *consent.Client.ClientId)
	}

	r := hClient.NewAcceptOAuth2ConsentRequest()
	r.SetGrantScope(consent.RequestedScope)
	r.SetGrantAccessTokenAudience(atAudience)
	r.SetSession(*session)
	r.SetRemember(true)

	ctx, span := s.tracer.Start(ctx, "hydra.OAuth2Api.AcceptOAuth2ConsentRequest")
	defer span.End()

	accept, res, err := s.hydra.OAuth2Api().AcceptOAuth2ConsentRequest(
		ctx,
	).ConsentChallenge(
		consent.GetChallenge(),
	).AcceptOAuth2ConsentRequest(
		*r,
	).Execute()

	if err != nil {
		// TODO @shipperizer we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", res)

		return nil, err
	}

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
