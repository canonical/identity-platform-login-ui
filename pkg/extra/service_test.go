package extra

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
)

//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_extra.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_kratos.go github.com/ory/kratos-client-go/v25 FrontendAPI
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestGetConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2APIGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest(challengeString)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return consent, new(http.Response), nil
		},
	)

	c, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

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
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	challengeString := "test.challenge"
	consentRequest := hClient.OAuth2APIGetOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.GetOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequest(ctx).Times(1).Return(consentRequest)
	mockHydraOAuth2API.EXPECT().GetOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIGetOAuth2ConsentRequestRequest) (*hClient.OAuth2ConsentRequest, *http.Response, error) {
			// use reflect as consentChallenge is a private attribute, also is a string pointer so need to cast it multiple times
			if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != challengeString {
				t.Fatalf("expected challenge string as %s, got %s", challengeString, *challenge)
			}

			return nil, new(http.Response), fmt.Errorf("error")
		},
	)

	c, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).GetConsent(ctx, challengeString)

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
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
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

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

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
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
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

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

	if a != nil {
		t.Fatalf("expected accept to be nil not  %v", a)
	}

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestAcceptConsentNoTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	// No tenant_id in consent context
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())
			sessionData := acceptReq.GetSession()
			// When no tenant_id is in the context, access_token should not have _tenant_id set
			accessTokenRaw := sessionData.GetAccessToken()
			if accessTokenRaw != nil {
				accessToken, _ := accessTokenRaw.(map[string]interface{})
				if _, ok := accessToken["_tenant_id"]; ok {
					t.Fatalf("expected no _tenant_id in access_token when consent context has none, got %v", accessToken)
				}
			}
			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "")

	if a != accept {
		t.Fatalf("expected accept to be %v not %v", accept, a)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}

func TestAcceptConsentPropagatesTenantIDFromContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOAuth2API := NewMockOAuth2API(ctrl)

	ctx := context.Background()
	redirect := "https://test.com/test"
	acceptRequest := hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest{
		ApiService: mockHydraOAuth2API,
	}
	// tenant_id in context (primary path — set by AcceptLoginRequest)
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.Context = map[string]interface{}{"tenant_id": "tenant-from-ctx"}
	identity := kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	accept := hClient.NewOAuth2RedirectTo(redirect)

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2API.AcceptOAuth2ConsentRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequest(ctx).Times(1).Return(acceptRequest)
	mockHydraOAuth2API.EXPECT().AcceptOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2APIAcceptOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			acceptReq := (*hClient.AcceptOAuth2ConsentRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2ConsentRequest").UnsafePointer())
			sessionData := acceptReq.GetSession()
			accessTokenRaw := sessionData.GetAccessToken()
			accessToken, ok := accessTokenRaw.(map[string]interface{})
			if !ok {
				t.Fatalf("expected access_token to be map[string]interface{}, got %T", accessTokenRaw)
			}
			tenantID, ok := accessToken["_tenant_id"]
			if !ok {
				t.Fatalf("expected _tenant_id in session.access_token, got %v", accessToken)
			}
			if tenantID != "tenant-from-ctx" {
				t.Fatalf("expected _tenant_id=tenant-from-ctx, got %v", tenantID)
			}
			return accept, new(http.Response), nil
		},
	)

	a, err := NewService(mockHydra, mockTracer, mockMonitor, mockLogger).AcceptConsent(ctx, *identity, consent, "tenant-from-ctx")

	if a != accept {
		t.Fatalf("expected accept to be %v not %v", accept, a)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
}
