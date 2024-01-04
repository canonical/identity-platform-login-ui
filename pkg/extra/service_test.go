package extra

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_extra.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_kratos.go github.com/ory/kratos-client-go FrontendApi
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_hydra.go github.com/ory/hydra-client-go/v2 OAuth2Api

func TestCheckSessionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookies = append(cookies, &http.Cookie{Name: "test", Value: "test"})
	session := kClient.NewSession("test", *kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"}))
	sessionRequest := kClient.FrontendApiToSessionRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.ToSession").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().ToSession(ctx).Times(1).Return(sessionRequest)
	mockKratosFrontendApi.EXPECT().ToSessionExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiToSessionRequest) (*kClient.Session, *http.Response, error) {
			// use reflect as cookie is a private attribute, also is a string pointer so need to cast it multiple times
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return session, new(http.Response), nil
		},
	)

	s, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

	if s != session {
		t.Fatalf("expected session to be %v not  %v", session, s)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCheckSessionFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookies = append(cookies, &http.Cookie{Name: "test", Value: "test"})
	sessionRequest := kClient.FrontendApiToSessionRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.ToSession").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().ToSession(ctx).Times(1).Return(sessionRequest)
	mockKratosFrontendApi.EXPECT().ToSessionExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiToSessionRequest) (*kClient.Session, *http.Response, error) {
			// use reflect as cookie is a private attribute, also is a string pointer so need to cast it multiple times
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	s, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

	if s != nil {
		t.Fatalf("expected session to be nil not  %v", s)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestGetConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2ApiGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2Api,
	}
	consent := hClient.NewOAuth2ConsentRequest(challengeString)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOAuth2Api)
	mockHydraOAuth2Api.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2Api.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2ApiGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return consent, new(http.Response), nil
		},
	)

	c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

	if c != consent {
		t.Fatalf("expected consent to be %v not  %v", consent, c)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}
func TestGetConsentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2ApiGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2Api,
	}
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOAuth2Api)
	mockHydraOAuth2Api.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2Api.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2ApiGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

	if c != nil {
		t.Fatalf("expected consent to be nil not  %v", c)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestAcceptConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2ApiAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2Api,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOAuth2Api)
	mockHydraOAuth2Api.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2Api.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2ApiAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != consent.GetChallenge() {
				t.Fatalf("expected challenge string as %s, got %s", consent.GetChallenge(), *challenge)
			}

			// use reflect as cookie is a private attribute, also is a pointer so need to cast it multiple times
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantScope(), consent.GetRequestedScope()) {
				t.Fatalf("expected scope as %s, got %s", consent.GetRequestedScope(), acceptReq.GetGrantScope())
			}

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), consent.GetRequestedAccessTokenAudience()) {
				t.Fatalf("expected audience as %s, got %s", consent.GetRequestedAccessTokenAudience(), acceptReq.GetGrantAccessTokenAudience())
			}

			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent)

	if a != accept {
		t.Fatalf("expected accept to be %v not  %v", accept, a)
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAcceptConsentFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2Api := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	acceptRequest := hClient.OAuth2ApiAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2Api,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOAuth2Api)
	mockHydraOAuth2Api.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2Api.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2ApiAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != consent.GetChallenge() {
				t.Fatalf("expected challenge string as %s, got %s", consent.GetChallenge(), *challenge)
			}

			// use reflect as cookie is a private attribute, also is a pointer so need to cast it multiple times
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())

			if !reflect.DeepEqual(acceptReq.GetGrantScope(), consent.GetRequestedScope()) {
				t.Fatalf("expected scope as %s, got %s", consent.GetRequestedScope(), acceptReq.GetGrantScope())
			}

			if !reflect.DeepEqual(acceptReq.GetGrantAccessTokenAudience(), consent.GetRequestedAccessTokenAudience()) {
				t.Fatalf("expected audience as %s, got %s", consent.GetRequestedAccessTokenAudience(), acceptReq.GetGrantAccessTokenAudience())
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	a, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent)

	if a != nil {
		t.Fatalf("expected accept to be nil not  %v", a)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}
