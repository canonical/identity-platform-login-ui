package kratos

import (
	"context"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	misc "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	kratos KratosClientInterface
	hydra  HydraClientInterface

	tracer  trace.Tracer
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

// We override the type from the kratos sdk, as it does not
// get marshalled correctly into json.
// For more info see: https://github.com/canonical/identity-platform-login-ui/pull/73/files#r1250460283
type ErrorBrowserLocationChangeRequired struct {
	Error *kClient.GenericError `json:"error,omitempty"`
	// Points to where to redirect the user to next.
	RedirectBrowserTo *string `json:"redirect_browser_to,omitempty"`
}

func (s *Service) CheckSession(ctx context.Context, cookies []*http.Cookie) (*kClient.Session, http.Header, error) {
	_, span := s.tracer.Start(ctx, "kratos.FrontendApi.ToSession")
	defer span.End()

	session, resp, err := s.kratos.FrontendApi().
		ToSession(ctx).
		Cookie(misc.CookiesToString(cookies)).
		Execute()

	if err != nil {
		// TODO @nsklikas we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", resp)

		return nil, nil, err
	}
	return session, resp.Header, nil
}

func (s *Service) AcceptLoginRequest(ctx context.Context, identityID string, lc string) (*hClient.OAuth2RedirectTo, http.Header, error) {
	_, span := s.tracer.Start(ctx, "hydra.OAuth2Api.AcceptOAuth2LoginRequest")
	defer span.End()

	accept := hClient.NewAcceptOAuth2LoginRequest(identityID)
	redirectTo, resp, err := s.hydra.OAuth2Api().
		AcceptOAuth2LoginRequest(ctx).
		LoginChallenge(lc).
		AcceptOAuth2LoginRequest(*accept).
		Execute()

	if err != nil {
		// TODO @nsklikas we shouldn't be logging this
		s.logger.Debugf("full HTTP response: %v", resp)
		return nil, nil, err
	}

	return redirectTo, resp.Header, nil
}

func (s *Service) CreateBrowserLoginFlow(
	ctx context.Context, aal, returnTo, loginChallenge string, refresh bool, cookies []*http.Cookie,
) (*kClient.LoginFlow, http.Header, error) {
	_, span := s.tracer.Start(ctx, "kratos.FrontendApi.CreateBrowserLoginFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		CreateBrowserLoginFlow(context.Background()).
		Aal(aal).
		ReturnTo(returnTo).
		LoginChallenge(loginChallenge).
		Refresh(refresh).
		Cookie(misc.CookiesToString(cookies)).
		Execute()
	if err != nil {
		s.logger.Debugf("full HTTP response: %v", resp)
		return nil, nil, err
	}

	return flow, resp.Header, nil
}

func (s *Service) GetLoginFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.LoginFlow, http.Header, error) {
	_, span := s.tracer.Start(ctx, "kratos.FrontendApi.GetLoginFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		GetLoginFlow(ctx).
		Id(id).
		Cookie(misc.CookiesToString(cookies)).
		Execute()
	if err != nil && resp.StatusCode != 422 {
		s.logger.Debugf("full HTTP response: %v", resp)
		return nil, nil, err
	}

	return flow, resp.Header, nil
}

func (s *Service) UpdateOIDCLoginFlow(
	ctx context.Context, flow string, body kClient.UpdateLoginFlowBody, cookies []*http.Cookie,
) (*ErrorBrowserLocationChangeRequired, http.Header, error) {
	_, span := s.tracer.Start(ctx, "kratos.FrontendApi.UpdateLoginFlow")
	defer span.End()

	_, resp, err := s.kratos.FrontendApi().
		UpdateLoginFlow(ctx).
		Flow(flow).
		UpdateLoginFlowBody(body).
		Cookie(misc.CookiesToString(cookies)).
		Execute()
	if err != nil && resp.StatusCode != 422 {
		s.logger.Debugf("full HTTP response: %v", resp)
		return nil, nil, err
	}

	redirectResp := new(ErrorBrowserLocationChangeRequired)
	err = misc.UnmarshalByteJson(resp.Body, redirectResp)
	if err != nil {
		s.logger.Debugf("Failed to unmarshal JSON: %s", err)
		return nil, nil, err
	}
	return redirectResp, resp.Header, nil
}

func (s *Service) GetFlowError(ctx context.Context, id string) (*kClient.FlowError, http.Header, error) {
	_, span := s.tracer.Start(ctx, "kratos.FrontendApi.GetFlowError")
	defer span.End()

	flowError, resp, err := s.kratos.FrontendApi().GetFlowError(context.Background()).Id(id).Execute()
	if err != nil {
		s.logger.Debugf("full HTTP response: %v", resp)
		return nil, nil, err
	}

	return flowError, resp.Header, nil
}

func (s *Service) ParseLoginFlowMethodBody(r *http.Request) (*kClient.UpdateLoginFlowBody, error) {
	body := new(kClient.UpdateLoginFlowWithOidcMethod)
	err := misc.ParseBody(r.Body, &body)
	if err != nil {
		return nil, err
	}
	ret := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(
		body,
	)
	return &ret, nil
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
