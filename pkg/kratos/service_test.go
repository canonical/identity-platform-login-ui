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
	gomock "github.com/golang/mock/gomock"
	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/otel/trace"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go github.com/ory/kratos-client-go FrontendApi
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_hydra.go github.com/ory/hydra-client-go/v2 OAuth2Api

func TestCheckSessionSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.ToSession").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	s, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

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
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendApi(ctrl)

	ctx := context.Background()
	cookies := make([]*http.Cookie, 0)
	cookies = append(cookies, &http.Cookie{Name: "test", Value: "test"})
	sessionRequest := kClient.FrontendApiToSessionRequest{
		ApiService: mockKratosFrontendApi,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.ToSession").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	s, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2LoginRequest").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	rt, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, identityID, loginChallenge)

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
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockHydraOauthApi := NewMockOAuth2Api(ctrl)

	ctx := context.Background()
	loginChallenge := "123456"
	identityID := "id"
	acceptLoginRequest := hClient.OAuth2ApiAcceptOAuth2LoginRequestRequest{
		ApiService: mockHydraOauthApi,
	}

	mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
	mockTracer.EXPECT().Start(ctx, "hydra.OAuth2Api.AcceptOAuth2LoginRequest").Times(1).Return(nil, trace.SpanFromContext(ctx))
	mockHydra.EXPECT().OAuth2Api().Times(1).Return(mockHydraOauthApi)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequest(ctx).Times(1).Return(acceptLoginRequest)
	mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequestExecute(gomock.Any()).Times(1).Return(nil, nil, fmt.Errorf("error"))

	rt, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, identityID, loginChallenge)

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
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.CreateBrowserLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CreateBrowserLoginFlow(ctx, aal, returnTo, loginChallenge, refresh, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.CreateBrowserLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).CreateBrowserLoginFlow(ctx, aal, returnTo, loginChallenge, refresh, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.GetLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	s, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetLoginFlow(ctx, id, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.GetLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetLoginFlow(ctx, id, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.UpdateLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).UpdateOIDCLoginFlow(ctx, flowId, *body, cookies)

	if !reflect.DeepEqual(*f, flow) {
		t.Fatalf("expected flow to be %+v not %+v", flow, *f)
	}
	if !reflect.DeepEqual(c, resp.Cookies()) {
		t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestUpdateLoginFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
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
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.UpdateLoginFlow").Times(1).Return(nil, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).UpdateOIDCLoginFlow(ctx, flowId, *body, cookies)

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
	mockTracer := NewMockTracer(ctrl)
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

	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.GetFlowError").Times(1).Return(nil, trace.SpanFromContext(ctx))
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

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetFlowError(ctx, id)

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
	mockTracer := NewMockTracer(ctrl)
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
	mockTracer.EXPECT().Start(ctx, "kratos.FrontendApi.GetFlowError").Times(1).Return(nil, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().GetFlowError(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().GetFlowErrorExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))

	f, c, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).GetFlowError(ctx, id)

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

func TestParseLoginFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockTracer := NewMockTracer(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(kClient.NewUpdateLoginFlowWithOidcMethodWithDefaults())
	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockHydra, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)
	if !reflect.DeepEqual(*b, body) {
		t.Fatalf("expected flow to be %+v not %+v", body, *b)
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}
