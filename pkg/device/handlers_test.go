// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/go-chi/chi/v5"
	hClient "github.com/ory/hydra-client-go/v26"
	"go.uber.org/mock/gomock"
)

func createGenericOpenAPIErr(body []byte) *hClient.GenericOpenAPIError {
	openAPIErr := &hClient.GenericOpenAPIError{}
	v := reflect.ValueOf(openAPIErr).Elem()
	f := v.FieldByName("body")
	rf := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	rf.Set(reflect.ValueOf(body))
	return openAPIErr
}

//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_device.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package device -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go

func TestHandleDeviceUserCodeAcceptSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	accept := hClient.NewOAuth2RedirectTo("test")

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(accept, nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	data, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	redirect := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if redirect.RedirectTo != accept.RedirectTo {
		t.Fatalf("expected %s, got %s.", accept.RedirectTo, redirect.RedirectTo)
	}
}

func TestHandleDeviceUserCodeAcceptParseUserCodeBodyFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleDeviceUserCodeAcceptAcceptUserCodeFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleDeviceUserCodeHydra404Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	hydraErrorPayload := `{"error":"not_found","error_description":"Unable to locate the request.","status_code":404}`
	err := createGenericOpenAPIErr([]byte(hydraErrorPayload))

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, err)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected HTTP status code 404 got %v", res.StatusCode)
	}

	data, errRead := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if errRead != nil {
		t.Fatalf("expected error to be nil got %v", errRead)
	}
	if string(data) != hydraErrorPayload {
		t.Fatalf("expected '%v' got '%v'", hydraErrorPayload, string(data))
	}
}

func TestHandleDeviceUserCodeHydra400Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	hydraErrorPayload := `{"error":"invalid_request","error_description":"The user_code provided is either invalid, expired or already used.","status_code":400}`
	err := createGenericOpenAPIErr([]byte(hydraErrorPayload))

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, err)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP status code 400 got %v", res.StatusCode)
	}

	data, errRead := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if errRead != nil {
		t.Fatalf("expected error to be nil got %v", errRead)
	}
	if string(data) != hydraErrorPayload {
		t.Fatalf("expected '%v' got '%v'", hydraErrorPayload, string(data))
	}
}

func TestHandleDeviceUserCodeHydra401Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	hydraErrorPayload := `{"error":"unauthorized","error_description":"Unauthorized.","status_code":401}`
	err := createGenericOpenAPIErr([]byte(hydraErrorPayload))

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, err)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP status code 401 got %v", res.StatusCode)
	}

	data, errRead := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if errRead != nil {
		t.Fatalf("expected error to be nil got %v", errRead)
	}
	if string(data) != hydraErrorPayload {
		t.Fatalf("expected '%v' got '%v'", hydraErrorPayload, string(data))
	}
}

func TestHandleDeviceUserCodeUnexpectedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	code := "ABCDEFGH"
	challenge := "7bb518c4eec2454dbb289f5fdb4c0ee2"

	userCodeRequest := hClient.NewAcceptDeviceUserCodeRequest()
	userCodeRequest.UserCode = &code
	jsonBody, _ := userCodeRequest.MarshalJSON()

	req := httptest.NewRequest(http.MethodPut, "/api/hydra/admin/oauth2/auth/requests/device/accept", io.NopCloser(bytes.NewBuffer(jsonBody)))
	values := req.URL.Query()
	values.Add("device_challenge", challenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseUserCodeBody(gomock.Any()).Return(userCodeRequest, nil)
	mockService.EXPECT().AcceptUserCode(gomock.Any(), challenge, userCodeRequest).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockTracer, mockLogger).RegisterEndpoints(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

