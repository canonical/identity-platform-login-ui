package kratos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go github.com/ory/kratos-client-go FrontendApi
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestCheckSessionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	session := kClient.NewSession("test", *kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"}))
	sessionRequest := kClient.FrontendApiToSessionRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.ToSession").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().ToSession(ctx).Times(1).Return(sessionRequest)
	mockKratosFrontendApi.EXPECT().ToSessionExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiToSessionRequest) (*kClient.Session, *http.Response, error) {
			// use reflect as cookie is a private attribute, also is a string pointer so need to cast it multiple times
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return session, &resp, nil
		},
	)

	s, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

	if s != session {
		t.Fatalf("expected session to be %v not  %v", session, s)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
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
	mockAuthz := NewMockAuthorizerInterface(ctrl)
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
	mockTracer.EXPECT().Start(ctx, "kratos.Service.ToSession").Times(1).Return(ctx, trace.SpanFromContext(ctx))
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

	s, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

	if s != nil {
		t.Fatalf("expected session to be nil not  %v", s)
	}
	if c != nil {
		t.Fatalf("expected cookies to be nil not  %v", c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestAcceptLoginRequestSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOauthApi := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	loginChallenge := "123456"
	identityID := "id"
	redirectTo := hClient.NewOAuth2RedirectTo("http://redirect/to/path")
	acceptLoginRequest := hClient.OAuth2ApiAcceptOAuth2LoginRequestRequest{
		ApiService: mockHydraOauthApi,
	}

	resp := new(http.Response)

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2LoginRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOauthApi)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequest(ctx).Times(1).Return(acceptLoginRequest)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r hClient.OAuth2ApiAcceptOAuth2LoginRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
			if lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer()); *lc != loginChallenge {
				t.Fatalf("expected loginChallenge to be %s, got %s", loginChallenge, *lc)
			}
			if id := (*hClient.AcceptOAuth2LoginRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2LoginRequest").UnsafePointer()); id.Subject != identityID {
				t.Fatalf("expected identityID to be %s, got %s", identityID, id.Subject)
			}
			return redirectTo, resp, nil
		},
	)

	rt, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, identityID, loginChallenge)

	if rt != redirectTo {
		t.Fatalf("expected redirect to be %v not  %v", redirectTo, rt)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAcceptLoginRequestFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOauthApi := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	loginChallenge := "123456"
	identityID := "id"
	acceptLoginRequest := hClient.OAuth2ApiAcceptOAuth2LoginRequestRequest{
		ApiService: mockHydraOauthApi,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2LoginRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOauthApi)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequest(ctx).Times(1).Return(acceptLoginRequest)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequestExecute(gomock.Any()).Times(1).Return(nil, nil, fmt.Errorf("error"))

	rt, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, identityID, loginChallenge)

	if rt != nil {
		t.Fatalf("expected redirect to be %v not  %v", nil, rt)
	}
	if c != nil {
		t.Fatalf("expected cookies to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestCreateBrowserLoginFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	aal := "aal"
	returnTo := "https://return/to/somewhere"
	loginChallenge := "123456"
	refresh := false
	flow := kClient.NewLoginFlowWithDefaults()
	request := kClient.FrontendApiCreateBrowserLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiCreateBrowserLoginFlowRequest) (*kClient.LoginFlow, *http.Response, error) {
			if _aal := (*string)(reflect.ValueOf(r).FieldByName("aal").UnsafePointer()); *_aal != aal {
				t.Fatalf("expected aal to be %s, got %s", aal, *_aal)
			}
			if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
				t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
			}
			if lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer()); *lc != loginChallenge {
				t.Fatalf("expected loginChallenge to be %s, got %s", loginChallenge, *lc)
			}
			if ref := (*bool)(reflect.ValueOf(r).FieldByName("refresh").UnsafePointer()); *ref != refresh {
				t.Fatalf("expected refresh to be %v, got %v", refresh, *ref)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return flow, &resp, nil
		},
	)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserLoginFlow(ctx, aal, returnTo, loginChallenge, refresh, cookies)

	if f != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, f)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCreateBrowserLoginFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	aal := "aal"
	returnTo := "https://return/to/somewhere"
	loginChallenge := "123456"
	refresh := false
	request := kClient.FrontendApiCreateBrowserLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserLoginFlow(ctx, aal, returnTo, loginChallenge, refresh, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected cookies to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestGetLoginFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"
	flow := kClient.NewLoginFlowWithDefaults()
	request := kClient.FrontendApiGetLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiGetLoginFlowRequest) (*kClient.LoginFlow, *http.Response, error) {
			if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
				t.Fatalf("expected id to be %s, got %s", id, *_id)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return flow, &resp, nil
		},
	)

	s, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetLoginFlow(ctx, id, cookies)

	if s != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, s)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetLoginFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"
	request := kClient.FrontendApiGetLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetLoginFlow(ctx, id, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestUpdateLoginFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	_redirectTo := "https://redirect/to/path"
	flow := ErrorBrowserLocationChangeRequired{
		RedirectBrowserTo: &_redirectTo,
	}
	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateLoginFlowBody)
	request := kClient.FrontendApiUpdateLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiUpdateLoginFlowRequest) (*ErrorBrowserLocationChangeRequired, *http.Response, error) {
			if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
				t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
			}
			if _body := (*kClient.UpdateLoginFlowBody)(reflect.ValueOf(r).FieldByName("updateLoginFlowBody").UnsafePointer()); *_body != *body {
				t.Fatalf("expected id to be %v, got %v", *body, *_body)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return &flow, &resp, nil
		},
	)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, *body, cookies)

	if *f.RedirectTo != *flow.RedirectBrowserTo {
		t.Fatalf("expected redirectTo to be %s not %s", *flow.RedirectBrowserTo, *f.RedirectTo)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestUpdateLoginFlowErrorWebAuthnNotSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	body := new(kClient.UpdateLoginFlowBody)

	request := kClient.FrontendApiUpdateLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	errorBody := &UiErrorMessages{
		Ui: kClient.UiContainer{
			Messages: []kClient.UiText{
				{
					Id: MissingSecurityKeySetup,
				},
			},
		},
	}
	errorBodyJson, _ := json.Marshal(errorBody)
	resp := http.Response{
		Body:       io.NopCloser(bytes.NewBuffer(errorBodyJson)),
		StatusCode: 400,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	_, _, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, *body, cookies)

	if err == nil {
		t.Fatalf("expected error not nil")
	}
	expectedError := fmt.Errorf("choose a different login method")
	if err.Error() != expectedError.Error() {
		t.Fatalf("expected error to be %v not %v", expectedError, err)
	}
}

func TestUpdateLoginFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	_redirectTo := "https://redirect/to/path"
	flow := ErrorBrowserLocationChangeRequired{
		RedirectBrowserTo: &_redirectTo,
	}
	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateLoginFlowBody)

	request := kClient.FrontendApiUpdateLoginFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, *body, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not %+v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestGetFlowErrorSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	id := "id"
	flow := kClient.NewFlowError(id)
	request := kClient.FrontendApiGetFlowErrorRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"K": []string{"V"}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetFlowError").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetFlowError(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetFlowErrorExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiGetFlowErrorRequest) (*kClient.FlowError, *http.Response, error) {
			if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
				t.Fatalf("expected id to be %s, got %s", id, *_id)
			}

			return flow, &resp, nil
		},
	)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetFlowError(ctx, id)

	if !reflect.DeepEqual(f, flow) {
		t.Fatalf("expected flow to be %+v not %+v", flow, f)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetFlowErrorFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	id := "id"
	request := kClient.FrontendApiGetFlowErrorRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"K": []string{"V"}},
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetFlowError").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetFlowError(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetFlowErrorExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetFlowError(ctx, id)

	if f != nil {
		t.Fatalf("expected flow to be %v not %+v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestCheckAllowedProviderAllowedSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	provider := "provider"
	oidcBody := kClient.NewUpdateLoginFlowWithOidcMethod("oidc", provider)
	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(oidcBody)

	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CheckAllowedProvider").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{provider}, nil)

	allowed, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CheckAllowedProvider(ctx, flow, &body)

	if !allowed {
		t.Fatalf("expected allowed to be true")
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCheckAllowedProviderNotAllowedSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	provider := "provider"
	oidcBody := kClient.NewUpdateLoginFlowWithOidcMethod("oidc", provider)
	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(oidcBody)

	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CheckAllowedProvider").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return([]string{"other_provider"}, nil)

	allowed, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CheckAllowedProvider(ctx, flow, &body)

	if allowed {
		t.Fatalf("expected allowed to be false")
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCheckAllowedProviderFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()
	provider := "provider"
	oidcBody := kClient.NewUpdateLoginFlowWithOidcMethod("oidc", provider)
	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(oidcBody)

	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CheckAllowedProvider").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(make([]string, 0), fmt.Errorf("oh no"))

	_, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CheckAllowedProvider(ctx, flow, &body)

	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestGetClientNameOathkeeper(t *testing.T) {
	loginFlow := &kClient.LoginFlow{}
	service := NewService(nil, nil, nil, nil, nil, nil)

	actualClientName := service.getClientName(loginFlow)

	const expectedClientName = ""
	if expectedClientName != actualClientName {
		t.Fatalf("Expected client name doesn't match")
	}
}

func TestGetClientNameOAuth2Request(t *testing.T) {
	expectedClientName := "mockClientName"
	loginFlow := &kClient.LoginFlow{Oauth2LoginRequest: &kClient.OAuth2LoginRequest{Client: &kClient.OAuth2Client{ClientName: &expectedClientName}}}
	service := NewService(nil, nil, nil, nil, nil, nil)

	actualClientName := service.getClientName(loginFlow)

	if expectedClientName != actualClientName {
		t.Fatalf("Expected client name doesn't match")
	}
}

func TestFilterFlowProviderListAllowAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	kratosProviders := []string{"1", "2", "3", "4"}
	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	ui := *kClient.NewUiContainerWithDefaults()
	kClient.NewUiNodeWithDefaults()
	for _, p := range kratosProviders {
		node := kClient.NewUiNodeWithDefaults()
		attributes := kClient.NewUiNodeInputAttributesWithDefaults()
		attributes.Value = p
		node.Attributes = kClient.UiNodeInputAttributesAsUiNodeAttributes(attributes)
		node.Group = "oidc"
		ui.Nodes = append(ui.Nodes, *node)
	}
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq
	flow.Ui = ui

	mockTracer.EXPECT().Start(ctx, "kratos.Service.FilterFlowProviderList").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(kratosProviders, nil)

	f, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).FilterFlowProviderList(ctx, flow)

	if !reflect.DeepEqual(f.Ui, ui) {
		t.Fatalf("expected ui to be %v not  %v", ui, f.Ui)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestFilterFlowProviderListAllowSome(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	kratosProviders := []string{"1", "2", "3", "4"}
	allowedProviders := []string{"1", "ab", "ba", "4"}
	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	ui := *kClient.NewUiContainerWithDefaults()
	kClient.NewUiNodeWithDefaults()
	for _, p := range kratosProviders {
		node := kClient.NewUiNodeWithDefaults()
		attributes := kClient.NewUiNodeInputAttributesWithDefaults()
		attributes.Value = p
		node.Attributes = kClient.UiNodeInputAttributesAsUiNodeAttributes(attributes)
		node.Group = "oidc"
		ui.Nodes = append(ui.Nodes, *node)
	}
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq
	flow.Ui = ui

	mockTracer.EXPECT().Start(ctx, "kratos.Service.FilterFlowProviderList").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allowedProviders, nil)

	f, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).FilterFlowProviderList(ctx, flow)

	expectedUi := *kClient.NewUiContainerWithDefaults()
	expectedUi.Nodes = []kClient.UiNode{ui.Nodes[0], ui.Nodes[3]}
	if !reflect.DeepEqual(f.Ui, expectedUi) {
		t.Fatalf("expected Ui to be %v not  %v", expectedUi, f.Ui)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestFilterFlowProviderListAllowNone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	kratosProviders := []string{"1", "2", "3", "4"}
	allowedProviders := []string{}
	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	ui := *kClient.NewUiContainerWithDefaults()
	kClient.NewUiNodeWithDefaults()
	for _, p := range kratosProviders {
		node := kClient.NewUiNodeWithDefaults()
		attributes := kClient.NewUiNodeInputAttributesWithDefaults()
		attributes.Value = p
		node.Attributes = kClient.UiNodeInputAttributesAsUiNodeAttributes(attributes)
		node.Group = "oidc"
		ui.Nodes = append(ui.Nodes, *node)
	}
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq
	flow.Ui = ui

	mockTracer.EXPECT().Start(ctx, "kratos.Service.FilterFlowProviderList").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(allowedProviders, nil)

	f, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).FilterFlowProviderList(ctx, flow)

	if !reflect.DeepEqual(f.Ui, ui) {
		t.Fatalf("expected Ui to be %v not  %v", ui, f.Ui)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestFilterFlowProviderListFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	ctx := context.Background()

	kratosProviders := []string{"1", "2", "3", "4"}
	client_name := "foo"
	client := kClient.NewOAuth2ClientWithDefaults()
	client.ClientName = &client_name
	loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
	loginReq.Client = client
	ui := *kClient.NewUiContainerWithDefaults()
	kClient.NewUiNodeWithDefaults()
	for _, p := range kratosProviders {
		node := kClient.NewUiNodeWithDefaults()
		attributes := kClient.NewUiNodeInputAttributesWithDefaults()
		attributes.Value = p
		node.Attributes = kClient.UiNodeInputAttributesAsUiNodeAttributes(attributes)
		node.Group = "oidc"
		ui.Nodes = append(ui.Nodes, *node)
	}
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Oauth2LoginRequest = loginReq
	flow.Ui = ui

	mockTracer.EXPECT().Start(ctx, "kratos.Service.FilterFlowProviderList").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, fmt.Errorf("oh no"))

	_, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).FilterFlowProviderList(ctx, flow)

	if err == nil {
		t.Fatalf("expected error to be not nil")
	}
}

func TestParseLoginFlowOidcMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(kClient.NewUpdateLoginFlowWithOidcMethodWithDefaults())
	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected flow to be %v not %v", expected, actual)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseLoginFlowPasswordMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateLoginFlowWithPasswordMethodWithDefaults()
	flow.SetMethod("password")

	body := kClient.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseLoginFlowTotpMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateLoginFlowWithTotpMethodWithDefaults()
	flow.SetMethod("totp")

	body := kClient.UpdateLoginFlowWithTotpMethodAsUpdateLoginFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseLoginFlowWebAuthnMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateLoginFlowWithWebAuthnMethodWithDefaults()
	flow.SetMethod("webauthn")

	body := kClient.UpdateLoginFlowWithWebAuthnMethodAsUpdateLoginFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetProviderNameWhenNotOidcMethod(t *testing.T) {
	loginFlow := &kClient.UpdateLoginFlowBody{}
	service := NewService(nil, nil, nil, nil, nil, nil)

	actualProviderName := service.getProviderName(loginFlow)

	expectedProviderName := ""
	if expectedProviderName != actualProviderName {
		t.Fatalf("Expected the provider to be %v, not %v", expectedProviderName, actualProviderName)
	}
}

func TestGetProviderNameOidc(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedProviderName := "someProvider"
	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateLoginFlowWithOidcMethod("", expectedProviderName)

	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(flow)
	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, _ := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

	actualProviderName := b.UpdateLoginFlowWithOidcMethod.Provider
	if expectedProviderName != actualProviderName {
		t.Fatalf("Expected the provider to be %v, not %v", expectedProviderName, actualProviderName)
	}
}

func TestParseRecoveryFlowCodeMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateRecoveryFlowWithCodeMethodWithDefaults()
	flow.SetMethod("code")

	body := kClient.UpdateRecoveryFlowWithCodeMethodAsUpdateRecoveryFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseRecoveryFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetRecoveryFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"
	flow := kClient.NewRecoveryFlowWithDefaults()
	request := kClient.FrontendApiGetRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiGetRecoveryFlowRequest) (*kClient.RecoveryFlow, *http.Response, error) {
			if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
				t.Fatalf("expected id to be %s, got %s", id, *_id)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return flow, &resp, nil
		},
	)

	s, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetRecoveryFlow(ctx, id, cookies)

	if s != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, s)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetRecoveryFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"
	request := kClient.FrontendApiGetRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetRecoveryFlow(ctx, id, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestCreateBrowserRecoveryFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	returnTo := "https://example.com/ui/reset_email"
	flow := kClient.NewRecoveryFlowWithDefaults()
	request := kClient.FrontendApiCreateBrowserRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlow(ctx).Times(1).Return(request)

	mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiCreateBrowserRecoveryFlowRequest) (*kClient.RecoveryFlow, *http.Response, error) {
			if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
				t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
			}

			return flow, &resp, nil
		},
	)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserRecoveryFlow(ctx, returnTo, cookies)

	if f != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, f)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCreateBrowserRecoveryFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	returnTo := "https://example.com/ui/reset_email"
	request := kClient.FrontendApiCreateBrowserRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserRecoveryFlow(ctx, returnTo, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected cookies to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestUpdateRecoveryFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	_redirectTo := "https://redirect/to/path"
	flow := ErrorBrowserLocationChangeRequired{
		RedirectBrowserTo: &_redirectTo,
	}
	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateRecoveryFlowBody)
	request := kClient.FrontendApiUpdateRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiUpdateRecoveryFlowRequest) (*ErrorBrowserLocationChangeRequired, *http.Response, error) {
			if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
				t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
			}
			if _body := (*kClient.UpdateRecoveryFlowBody)(reflect.ValueOf(r).FieldByName("updateRecoveryFlowBody").UnsafePointer()); *_body != *body {
				t.Fatalf("expected id to be %v, got %v", *body, *_body)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return &flow, &resp, nil
		},
	)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateRecoveryFlow(ctx, flowId, *body, cookies)

	if *f.RedirectTo != *flow.RedirectBrowserTo {
		t.Fatalf("expected redirectTo to be %s not %s", *flow.RedirectBrowserTo, *f.RedirectTo)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestUpdateRecoveryFlowFailOnUpdateRecoveryFlowExecute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	_redirectTo := "https://redirect/to/path"
	flow := ErrorBrowserLocationChangeRequired{
		RedirectBrowserTo: &_redirectTo,
	}
	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateRecoveryFlowBody)

	request := kClient.FrontendApiUpdateRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateRecoveryFlow(ctx, flowId, *body, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not %+v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestUpdateRecoveryFlowFailOnInvalidRecoveryCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"
	flow := &kClient.RecoveryFlow{
		Ui: kClient.UiContainer{
			Messages: []kClient.UiText{
				{
					Id: InvalidRecoveryCode,
				},
			},
		},
	}

	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateRecoveryFlowBody)

	request := kClient.FrontendApiUpdateRecoveryFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header:     http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:       io.NopCloser(bytes.NewBuffer(flowJson)),
		StatusCode: 200,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(flow, &resp, nil)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateRecoveryFlow(ctx, flowId, *body, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not %+v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not  %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
	expectedError := fmt.Errorf("the recovery code is invalid or has already been used")
	if err.Error() != expectedError.Error() {
		t.Fatalf("expected error to be %v not %v", expectedError, err)
	}
}

func TestParseSettingsFlowPasswordMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateSettingsFlowWithPasswordMethodWithDefaults()
	flow.SetMethod("password")

	body := kClient.UpdateSettingsFlowWithPasswordMethodAsUpdateSettingsFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseSettingsFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseSettingsFlowTotpMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateSettingsFlowWithTotpMethodWithDefaults()
	flow.SetMethod("totp")

	body := kClient.UpdateSettingsFlowWithTotpMethodAsUpdateSettingsFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseSettingsFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestParseSettingsFlowWebAuthnMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateSettingsFlowWithWebAuthnMethodWithDefaults()
	flow.SetMethod("webauthn")

	body := kClient.UpdateSettingsFlowWithWebAuthnMethodAsUpdateSettingsFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).ParseSettingsFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetSettingsFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"

	flow := kClient.NewSettingsFlowWithDefaults()
	request := kClient.FrontendApiGetSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetSettingsFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiGetSettingsFlowRequest) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error) {
			if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
				t.Fatalf("expected id to be %s, got %s", id, *_id)
			}

			return flow, nil, nil
		},
	)

	s, r, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetSettingsFlow(ctx, id, cookies)

	if s != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, s)
	}
	if r != nil {
		t.Fatalf("expected response to be nil not  %v", r)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetSettingsFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	id := "id"
	request := kClient.FrontendApiGetSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.GetSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetSettingsFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, r, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).GetSettingsFlow(ctx, id, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if r != nil {
		t.Fatalf("expected response to be %v not  %v", nil, r)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestCreateBrowserSettingsFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	returnTo := "https://example.com/ui/reset_complete"
	flow := kClient.NewSettingsFlowWithDefaults()
	request := kClient.FrontendApiCreateBrowserSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlow(ctx).Times(1).Return(request)

	mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiCreateBrowserSettingsFlowRequest) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error) {
			if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
				t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
			}

			return flow, nil, nil
		},
	)

	f, r, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserSettingsFlow(ctx, returnTo, cookies)

	if f != flow {
		t.Fatalf("expected flow to be %v not  %v", flow, f)
	}
	if r != nil {
		t.Fatalf("expected response to be nil not %v", r)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestCreateBrowserSettingsFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	returnTo := "https://example.com/ui/reset_complete"
	request := kClient.FrontendApiCreateBrowserSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		StatusCode: http.StatusNotFound,
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf(""))
	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)

	f, r, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).CreateBrowserSettingsFlow(ctx, returnTo, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not  %v", nil, f)
	}
	if r != nil {
		t.Fatalf("expected response to be %v not  %v", nil, r)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestUpdateSettingsFlowSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"

	flow := kClient.NewSettingsFlowWithDefaults()

	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateSettingsFlowBody)
	request := kClient.FrontendApiUpdateSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateSettingsFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendApiUpdateSettingsFlowRequest) (*kClient.SettingsFlow, *http.Response, error) {
			if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
				t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
			}
			if _body := (*kClient.UpdateSettingsFlowBody)(reflect.ValueOf(r).FieldByName("updateSettingsFlowBody").UnsafePointer()); *_body != *body {
				t.Fatalf("expected id to be %v, got %v", *body, *_body)
			}
			if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
				t.Fatalf("expected cookie string as test=test, got %s", *cookie)
			}

			return flow, &resp, nil
		},
	)

	_, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateSettingsFlow(ctx, flowId, *body, cookies)

	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestUpdateSettingsFlowFailOnUpdateSettingsFlowExecute(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookie := &http.Cookie{Name: "test", Value: "test"}
	cookies = append(cookies, cookie)
	flowId := "flow"

	flow := kClient.NewSettingsFlowWithDefaults()
	flowJson, _ := json.Marshal(flow)
	body := new(kClient.UpdateSettingsFlowBody)

	request := kClient.FrontendApiUpdateSettingsFlowRequest{
		ApiService: mockKratosFrontendApi,
	}
	resp := http.Response{
		Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
		Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateSettingsFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	f, c, err := NewService(mockKratos, mockHydra, mockAuthz, mockTracer, mockMonitor, mockLogger).UpdateSettingsFlow(ctx, flowId, *body, cookies)

	if f != nil {
		t.Fatalf("expected flow to be %v not %+v", nil, f)
	}
	if c != nil {
		t.Fatalf("expected header to be %v not %v", nil, c)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}
