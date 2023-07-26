package extra

import (
	"context"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	misc "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"go.opentelemetry.io/otel/trace"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

type Service struct {
	kratos KratosClientInterface
	hydra  HydraClientInterface

	tracer  trace.Tracer
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (s *Service) CheckSession(ctx context.Context, cookies []*http.Cookie) (*kClient.Session, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.FrontendApi.ToSession")
	defer span.End()

	session, r, err := s.kratos.FrontendApi().ToSession(
		ctx,
	).Cookie(
		misc.CookiesToString(cookies),
	).Execute()

	if err != nil {
		// TODO @shipperizer we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", r)

		return nil, err
	}
	return session, nil
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

	r := hClient.NewAcceptOAuth2ConsentRequest()
	r.SetGrantScope(consent.RequestedScope)
	r.SetGrantAccessTokenAudience(consent.RequestedAccessTokenAudience)
	r.SetSession(*session)

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

func NewService(kratos KratosClientInterface, hydra HydraClientInterface, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.kratos = kratos
	s.hydra = hydra

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}
